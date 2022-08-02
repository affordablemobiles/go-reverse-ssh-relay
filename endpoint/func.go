package main

import (
	"encoding/base64"
	"fmt"
	"os"
)

func processConfig() error {
	if serverKey, err := base64.StdEncoding.DecodeString(globalConfig.ServerKeyEncoded); err != nil {
		return fmt.Errorf("Unable to BASE64 decode server key")
	} else {
		globalConfig.serverKey = string(serverKey)
	}

	/*for i, kenc := range globalConfig.AllowedClientsEncoded {
		if clientKey, err := base64.StdEncoding.DecodeString(kenc); err != nil {
			return fmt.Errorf("Unable to BASE64 decode client key: %d", i)
		} else {
			globalConfig.allowedClients = append(globalConfig.allowedClients, string(clientKey))
		}
	}*/

	globalConfig.allowedClients = globalConfig.AllowedClientsEncoded

	return nil
}

func GetRunningIdentity() (string, string, string, string, error) {
	data := map[string]string{
		"GOOGLE_CLOUD_PROJECT": os.Getenv("GOOGLE_CLOUD_PROJECT"),
		"GAE_SERVICE":          os.Getenv("GAE_SERVICE"),
		"GAE_VERSION":          os.Getenv("GAE_VERSION"),
		"GAE_INSTANCE":         os.Getenv("GAE_INSTANCE"),
	}

	for k, v := range data {
		if v == "" {
			return "", "", "", "", fmt.Errorf("GetRunningIdentity: Environment Value Empty: %s", k)
		}
	}

	return data["GOOGLE_CLOUD_PROJECT"], data["GAE_SERVICE"], data["GAE_VERSION"], data["GAE_INSTANCE"], nil
}
