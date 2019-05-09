package scwconfig

import (
	"fmt"
	"io/ioutil"
	"os"
)

// LoadWithProfile call Load() and set withProfile with the profile name.
func LoadWithProfile(profileName string) (Config, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}

	v2Loaded := config.(*configV2)
	v2Loaded.withProfile = profileName
	return v2Loaded.catchInvalidProfile()
}

// Load config in the following order:
// - config file from SCW_CONFIG_PATH (V2 or V1)
// - config file V2
// - config file V1
// When the latest is found it migrates the V1 config
// to a V2 config following the V2 config path.
func Load() (Config, error) {
	// STEP 1: try to load config file from SCW_CONFIG_PATH
	configPath := os.Getenv(scwConfigPathEnv)
	if configPath != "" {
		content, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read $%s: %s", scwConfigPathEnv, err)
		}
		confV1, err := unmarshalConfV1(content)
		if err == nil {
			return confV1.toV2().catchInvalidProfile()
		}
		confV2, err := unmarshalConfV2(content)
		if err != nil {
			return nil, fmt.Errorf("content of $%s (%s) is invalid: %s", scwConfigPathEnv, configPath, err)
		}

		return confV2.catchInvalidProfile()
	}

	// STEP 2: try to load config file V2
	v2Path, v2PathOk := GetConfigV2FilePath()
	if v2PathOk && fileExist(v2Path) {
		file, err := ioutil.ReadFile(v2Path)
		if err != nil {
			return nil, fmt.Errorf("cannot read config file: %s", err)
		}

		confV2, err := unmarshalConfV2(file)
		if err != nil {
			return nil, fmt.Errorf("content of config file %s is invalid: %s", v2Path, err)
		}
		return confV2.catchInvalidProfile()
	}

	// STEP 3: try to load config file V1
	v1Path, v1PathOk := GetConfigV1FilePath()
	if !v1PathOk {
		return (&configV2{}).catchInvalidProfile()
	}
	file, err := ioutil.ReadFile(v1Path)
	if err != nil {
		return (&configV2{}).catchInvalidProfile() // ignore if file doesn't exist
	}
	confV1, err := unmarshalConfV1(file)
	if err != nil {
		return nil, fmt.Errorf("content of config file %s is invalid json: %s", v1Path, err)
	}

	// STEP 4: migrate V1 config to V2 config file
	if v2PathOk {
		err = migrateV1toV2(confV1, v2Path)
		if err != nil {
			return nil, err
		}
	}

	return confV1.toV2().catchInvalidProfile()
}

func fileExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}