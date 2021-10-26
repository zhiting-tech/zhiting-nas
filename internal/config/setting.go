package config

import "github.com/spf13/viper"

func NewSetting() (*Setting, error) {
	vp := viper.New()
	vp.SetConfigName("app")
	vp.AddConfigPath("./")
	vp.SetConfigType("yaml")
	err := vp.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return &Setting{vp: vp}, nil
}

type Setting struct {
	vp *viper.Viper
}

func (s *Setting) ReadSection(k string, v interface{}) error {
	err := s.vp.UnmarshalKey(k, v)
	if err != nil {
		return err
	}

	return nil
}