package setting

type ClusterSettingS struct {
	ConfigPath   string
	NetInterface string
}

type AppSettingS struct {
	LogSavePath string
	LogFileName string
	LogFileExt  string
}

type DatabaseSettingS struct {
	DBType string
	Host   string
	Token  string
	Org    string
	Bucket string
}

var sections = make(map[string]interface{})

func (s *Setting) ReadSection(k string, v interface{}) error {
	err := s.vp.UnmarshalKey(k, v)
	if err != nil {
		return err
	}

	if _, ok := sections[k]; !ok {
		sections[k] = v
	}
	return nil
}

func (s *Setting) ReloadAllSection() error {
	for k, v := range sections {
		err := s.ReadSection(k, v)
		if err != nil {
			return err
		}
	}

	return nil
}
