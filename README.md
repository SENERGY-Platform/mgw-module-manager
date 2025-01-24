mgw-module-manager
=======

Generate swagger docs:

    swag init -g routes.go -o handler/http_hdl/swagger_docs -dir handler/http_hdl/standard,handler/http_hdl/shared --parseDependency --instanceName standard
    swag init -g routes.go -o handler/http_hdl/swagger_docs -dir handler/http_hdl/restricted,handler/http_hdl/shared --parseDependency --instanceName restricted