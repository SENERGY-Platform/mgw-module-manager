module github.com/SENERGY-Platform/mgw-module-manager

go 1.20

require (
	github.com/SENERGY-Platform/gin-middleware v0.3.0
	github.com/SENERGY-Platform/go-cc-job-handler v0.1.0
	github.com/SENERGY-Platform/go-service-base v0.8.0
	github.com/SENERGY-Platform/mgw-container-engine-wrapper/client v0.6.0
	github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib v0.9.0
	github.com/SENERGY-Platform/mgw-host-manager/client v0.1.0
	github.com/SENERGY-Platform/mgw-host-manager/lib v0.1.3
	github.com/SENERGY-Platform/mgw-modfile-lib v0.11.0
	github.com/SENERGY-Platform/mgw-module-lib v0.10.0
	github.com/SENERGY-Platform/mgw-module-manager/lib v0.0.0-00010101000000-000000000000
	github.com/SENERGY-Platform/mgw-secret-manager/pkg v0.1.2
	github.com/gin-contrib/requestid v0.0.6
	github.com/gin-gonic/gin v1.9.1
	github.com/go-git/go-git/v5 v5.7.0
	github.com/go-sql-driver/mysql v1.7.0
	github.com/google/uuid v1.3.0
	github.com/y-du/go-log-level v0.2.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20230612151507-41ef4d1f67a4 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230518184743-7afd39499903 // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.4.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/skeema/knownhosts v1.1.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/y-du/go-env-loader v0.5.0 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

replace github.com/SENERGY-Platform/mgw-module-manager/lib => ./lib
