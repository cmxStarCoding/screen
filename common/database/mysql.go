package database

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
)

var (
	CrmDB *gorm.DB
	CmsDB *gorm.DB
	MapiReportDB *gorm.DB
)

// InitCrmDB 初始化数据库连接
func InitCrmDB() {

	//database := projectConfig["db_database"]
	//host := projectConfig["db_host"]
	//username := projectConfig["db_username"]
	//password := projectConfig["db_password"]
	//port := projectConfig["db_port"]

	viper.SetConfigFile("../common/config.ini")
	viper.ReadInConfig()

	database := viper.GetString("crm_db.database")
	host := viper.GetString("crm_db.host")
	username := viper.GetString("crm_db.username")
	password := viper.GetString("crm_db.password")
	port := viper.GetString("crm_db.port")

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"
	var connectorErr error
	CrmDB, connectorErr = gorm.Open(mysql.New(mysql.Config{
		DSN:                     dsn,
		DontSupportRenameColumn: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			//TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if connectorErr != nil {
		log.Fatalf("Failed to connect to database: %s", fmt.Sprintf("%v", connectorErr))
	}

	// 设置连接池配置（可选）
	sqlDB, err := CrmDB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: ", fmt.Sprintf("%v", connectorErr.Error()))
	}

	// 设置连接池大小等配置（根据实际情况进行调整）
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
}

// InitCmsDB 初始化数据库连接
func InitCmsDB() {

	//database := projectConfig["db_database"]
	//host := projectConfig["db_host"]
	//username := projectConfig["db_username"]
	//password := projectConfig["db_password"]
	//port := projectConfig["db_port"]

	viper.SetConfigFile("../common/config.ini")
	viper.ReadInConfig()

	database := viper.GetString("cms_db.database")
	host := viper.GetString("cms_db.host")
	username := viper.GetString("cms_db.username")
	password := viper.GetString("cms_db.password")
	port := viper.GetString("cms_db.port")

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"
	var connectorErr error
	CmsDB, connectorErr = gorm.Open(mysql.New(mysql.Config{
		DSN:                     dsn,
		DontSupportRenameColumn: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			//TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if connectorErr != nil {
		log.Fatalf("Failed to connect to database: %s", fmt.Sprintf("%v", connectorErr))
	}

	// 设置连接池配置（可选）
	sqlDB, err := CmsDB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: ", fmt.Sprintf("%v", connectorErr.Error()))
	}

	// 设置连接池大小等配置（根据实际情况进行调整）
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
}


// InitMapiReportDB 初始化数据库连接
func InitMapiReportDB() {

	//database := projectConfig["db_database"]
	//host := projectConfig["db_host"]
	//username := projectConfig["db_username"]
	//password := projectConfig["db_password"]
	//port := projectConfig["db_port"]

	viper.SetConfigFile("../common/config.ini")
	viper.ReadInConfig()

	database := viper.GetString("mapi_report_db.database")
	host := viper.GetString("mapi_report_db.host")
	username := viper.GetString("mapi_report_db.username")
	password := viper.GetString("mapi_report_db.password")
	port := viper.GetString("mapi_report_db.port")

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + database + "?charset=utf8mb4&parseTime=True&loc=Local"
	var connectorErr error
	MapiReportDB, connectorErr = gorm.Open(mysql.New(mysql.Config{
		DSN:                     dsn,
		DontSupportRenameColumn: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			//TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	if connectorErr != nil {
		log.Fatalf("Failed to connect to database: %s", fmt.Sprintf("%v", connectorErr))
	}

	// 设置连接池配置（可选）
	sqlDB, err := MapiReportDB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: ", fmt.Sprintf("%v", connectorErr.Error()))
	}

	// 设置连接池大小等配置（根据实际情况进行调整）
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
}