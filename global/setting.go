package global

import (
	"kubebpfbox/pkg/logger"
	"kubebpfbox/pkg/setting"
)

var (
	ClusterSetting  *setting.ClusterSettingS
	AppSetting      *setting.AppSettingS
	DatabaseSetting *setting.DatabaseSettingS
	Logger          *logger.Logger
)
