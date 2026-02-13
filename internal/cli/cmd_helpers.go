package cli

import "errors"

func requireApp(getApp func() *App) (*App, error) {
	app := getApp()
	if app == nil {
		return nil, errors.New("app not initialized")
	}
	return app, nil
}
