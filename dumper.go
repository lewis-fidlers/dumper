package main

import (
	"fmt"
	"log"
	"strings"

	"flag"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"

	"path/filepath"
	"regexp"
	"time"
)

func green(in string) (out string) {
	in = "\033[32m" + in + "\033[0m"
	return in
}

func red(in string) (out string) {
	in = "\033[33m" + in + "\033[0m"
	return in
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <environment>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func sqlite3_dump(l interface{}, name string) {
	fmt.Printf("sqlite3 db/development.sqlite3 .dump > dump")
}

func mysql_dump(config DbConfig, name string) (out string){
	command := fmt.Sprintf("mysqldump -u %s -p -h %s %s > %s.sql", config.Username, config.Host, config.Database, name)
	if len(config.Password) > 0 {
		command = fmt.Sprintf("Password: %s\n\n%s", config.Password, command)
	}
	return command
}

func mysql_restore(config DbConfig, name string) (out string){
	command := fmt.Sprintf("mysql -u %s -p -h %s %s < %s.sql", config.Username, config.Host, config.Database, name)
	if len(config.Password) > 0 {
		command = fmt.Sprintf("Password: %s\n\n%s", config.Password, command)
	}
	return command
}

func fetch(l map[string]interface{}, key string, value interface{}) (out interface{}) {
	if val, ok := l[key]; ok {
		return val
	} else {
		return value
	}
}


func pg_dump(config DbConfig, name string) (out string) {
	username := config.Username
	hostname := config.Host
	command := fmt.Sprintf("pg_dump -Fc --no-acl --no-owner --clean -U %s -h %s %s > %s.dump", username, hostname, config.Database, name)
	if len(config.Password) > 0 {
		command = fmt.Sprintf("PGPASSWORD=%s %s", config.Password, command)
	}
	return command
}

func pg_restore(config DbConfig, name string) (out string){
	username := config.Username
	hostname := config.Host
	command := fmt.Sprintf("pg_restore --verbose --clean --no-acl --no-owner -h %s -U %s -d %s %s.dump", hostname, username, config.Database, name)
	if len(config.Password) > 0 {
		command = fmt.Sprintf("PGPASSWORD=%s %s", config.Password, command)
	}
	return command
}

func get_environment(argument string) (environment string){
	environment = "development"
	if strings.TrimSpace(argument) != "" {
		environment = strings.TrimSpace(argument)
	}
	return environment
}

func (self *DbConfig) SetDefaults() {
	if self.Host == "" {
		self.Host = "localhost"
	}
}

func get_config(yamlData []byte, environment string) (config DbConfig, err error){
	m := make(map[string]DbConfig)
	err = yaml.Unmarshal(yamlData, &m)
	if err != nil {
		return m[environment], err
	}
	config, ok := m[environment]
	if !ok {
		var keys[]string
		for key, _ := range m {
			keys = append(keys, key)
		}
		err := fmt.Errorf("No such environment found. Use one of: %s", keys)
		return m[environment], err
	}
	config.SetDefaults()
	return config, nil
}

var currentDir = func() string {
	p,_ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}

var file_exists = func(path string) (err error) {
	_, err = os.Stat(path)
	return err
}

func get_yaml_path (path string) (out string, err error) {
	if path == "" {
		path = currentDir()
	}
	if filepath.Ext(path) != ".yml" {
		path = filepath.Join(path, "config", "database.yml")
	}
	err = file_exists(path)
	return path, err
}

type DbConfig struct {
	Adapter string
	Host string
	Database string
	Username string
	Password string
}

func main() {
	flag.Usage = usage
	var force bool
	var path string

	flag.BoolVar(&force, "F", false, "Show restore operation")
	flag.StringVar(&path, "p", "", "Path to yaml (otherwise config/database.yml")
	flag.Parse()

	environment := get_environment(flag.Arg(0))
	yamlFile, err := get_yaml_path(path)
	if err != nil {
		log.Fatalf("No Yaml file found: %v", err)
	}
	name := fmt.Sprintf("%s_%s_%s", filepath.Base(filepath.Dir(filepath.Dir(yamlFile))), environment[0:3], time.Now().Format("20060102"))
	yamlData, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		log.Fatalf("Error Reading Yaml: %v", err)
	}
	config, err := get_config(yamlData, environment)
	if err != nil {
		log.Fatalf("Error Parsing Yaml: %v", err)
	}
	adapter := config.Adapter
	pg := regexp.MustCompile("postgres")
	mysql := regexp.MustCompile("mysql")
	foundPg := pg.MatchString(adapter)
	foundMysql := mysql.MatchString(adapter)

	fmt.Printf("%s\n\n", green("Dump:"))
	if foundPg {
		dump := pg_dump(config, name)
		fmt.Printf("%s\n", green(dump))
		if force {
			fmt.Printf("\n%s\n\n", red("Restore:"))
			fmt.Printf("%s", pg_restore(config, name))
		}
	}
	if foundMysql {
		fmt.Printf("%s\n", mysql_dump(config, name))
		if force {
			fmt.Printf("\n\n%s\n\n", red("Restore:"))
			mysql_restore(config, name)
		}
	}
}
