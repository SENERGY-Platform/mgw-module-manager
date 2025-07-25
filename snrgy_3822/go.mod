module github.com/SENERGY-Platform/mgw-module-manager

go 1.24.2

require (
	github.com/SENERGY-Platform/gin-middleware v0.9.0
	github.com/SENERGY-Platform/go-service-base/config-hdl v1.2.0
	github.com/SENERGY-Platform/go-service-base/srv-info-hdl v0.2.0
	github.com/SENERGY-Platform/go-service-base/struct-logger v0.4.1
	github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib v0.17.0
	github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib v0.1.1
	github.com/SENERGY-Platform/mgw-modfile-lib v0.20.0
	github.com/SENERGY-Platform/mgw-module-lib v0.23.0
	github.com/SENERGY-Platform/mgw-module-manager/lib v0.0.0-00000000000000-000000000000
	github.com/gin-contrib/requestid v1.0.5
	github.com/gin-gonic/gin v1.10.1
	github.com/google/uuid v1.6.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	code.cloudfoundry.org/bytefmt v0.31.0 // indirect
	github.com/SENERGY-Platform/go-env-loader v0.5.3 // indirect
	github.com/bytedance/sonic v1.13.2 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.0.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.26.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.15.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/SENERGY-Platform/mgw-module-manager/lib => ./lib
