module github.com/choopm/stdfx

go 1.23

replace github.com/choopm/stdfx/examples/everything => ./examples/everything

require (
	github.com/choopm/stdfx/examples/everything v0.0.0
	github.com/creasty/defaults v1.8.0
	github.com/earthboundkid/versioninfo/v2 v2.24.1
	github.com/go-viper/mapstructure/v2 v2.2.1
	github.com/rs/zerolog v1.33.0
	github.com/samber/slog-zap/v2 v2.6.1
	github.com/samber/slog-zerolog/v2 v2.7.2
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.20.0-alpha.6
	github.com/stretchr/testify v1.10.0
	github.com/xhit/go-str2duration/v2 v2.1.0
	go.uber.org/fx v1.23.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.10.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/samber/slog-common v0.18.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)