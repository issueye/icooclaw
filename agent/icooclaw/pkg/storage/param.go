package storage

// ParamConfig 运行时参数配置模型
type ParamConfig struct {
	Model
	Key         string `gorm:"column:key;type:varchar(100);not null;comment:参数键" json:"key"`                       // 参数键
	Value       string `gorm:"column:value;type:text;comment:参数值(JSON格式)" json:"value"`                           // 参数值（JSON 格式）
	Description string `gorm:"column:description;type:varchar(500);comment:参数描述" json:"description"`               // 参数描述
	Group       string `gorm:"column:group;type:varchar(50);default:'general';comment:参数分组" json:"group"`          // 参数分组
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`           // 是否启用
}

// TableName returns the table name for ParamConfig.
func (ParamConfig) TableName() string {
	return tableNamePrefix + "param_config"
}
