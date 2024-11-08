## ci.sh basic usage

```bash
bash ci.sh (print help)
bash ci.sh test (running test stage)
bash ci.sh v|version (check version)
bash ci.sh build [$cmd_path] [$env_name] [no-push <default push> - optional]
```

Available command : v|version, test, build, build-base.

## Export ci .env config
```bash
export CI_CONFIG_DIR=.env
export $(grep -v '^#' .env | xargs)
```

## Build and push image
```bash
bash ci.sh build cmd/service/location develop
```

## Build image without pushing the image to registry
```bash
bash ci.sh build cmd/graphql develop no-push
```

## Base image
Change config on .env to force build base image. Default is automatic detect changes in go.mod and go.sum files

## Misc
Image tag format : {app_name}.{env_name}-{commit}-{date}
e.g : ```graphql.develop-26a24ee-211102200851```