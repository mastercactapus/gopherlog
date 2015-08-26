package logstash

type Config struct {
	Input  []PluginConfig
	Filter []PluginConfig
	Output []PluginConfig
}
type PluginConfig struct {
	Name   string
	Fields PluginFields
}
type PluginFields map[string]interface{}
