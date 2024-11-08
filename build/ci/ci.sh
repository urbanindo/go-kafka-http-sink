#!/bin/bash

#Abort script at first error
# set -ex

_main() {
    _init
    _set_env_name
    case $CI_STAGE in
        version|v)
            _version
            ;;
        test)
            _init_test
            _run_lint
            _run_test
            ;;
        unshallow-git)
            _unshallow_git
            ;;
        build-base)
            _build_base
            ;;
        build)
            _verify_build
            # _parallel_limit_start
            _build
            # _parallel_limit_done
            ;;
        *)
            _usage
            ;;
    esac
}

# This function cannot ensure the real parallel locking.
# It may passing over than limit that we set.
_parallel_limit_start() {
    if [[ -z "$PARALLEL_LIMIT" ]]; then
        PARALLEL_LIMIT=5
    fi

    _current_time=$(date)
    set -e
    CURRENT_WORKER=$(echo -n "$_current_time::$RANDOM" | md5sum | cut -d ' ' -f 1)
    echo "Current worker: $CURRENT_WORKER"
    mkdir -p $WORKSPACE/tmp/parallel/
    set +e

    # WORKSPACE
    while :
    do
        _total_files=$(ls -1q $WORKSPACE/tmp/parallel/ | wc -l)
        if (( $_total_files < $PARALLEL_LIMIT )); then
            break
        fi
        
        echo "wait for other job(s) to finish. current parallel: $_total_files"
        sleep 2
    done

    echo "1" > $WORKSPACE/tmp/parallel/$CURRENT_WORKER
}

_parallel_limit_done() {
    rm "$WORKSPACE/tmp/parallel/$CURRENT_WORKER"
}

_set_env_name() {
    if [[ "$BRANCH_NAME" == master ]]; then
        ENV_NAME=prod
    elif [[ "$BRANCH_NAME" == staging ]] || [[ "$BRANCH_NAME" == hotfix* ]]; then
        ENV_NAME=staging
    elif [[ "$BRANCH_NAME" == develop ]]; then
        ENV_NAME=develop
    else
        ENV_NAME=$BRANCH_NAME-develop
    fi
}

_validation() {
    if [[ -z $CI_STAGE ]] || [[ -z $CMD_PATH ]] || [[ -z $BRANCH_NAME ]];
        then
            echo "[error] -- not enough arguments"
            echo "Usage : $0 build CMD_PATH ENV_NAME"
            exit 1
    fi
}

_verify_build_base() {
    if echo "$COMMIT_MESSAGE" | grep -qE "\(build_base\)"; then
        echo "[info] -- commit message is forcing to build_base"
        return 0
    fi
    echo "[info] -- commit message command to skip build_base"
    exit 0
}

_verify_build() {
    if echo "$COMMIT_MESSAGE" | grep -qE "\(build_all\)"; then
        echo "[info] -- exec build_all"
    elif ! echo "$COMMIT_MESSAGE" | grep -qE "\($COMMIT_PARSE\)"; then
        echo "[info] -- commit message is not valid for '\($COMMIT_PARSE\)'. skip..."
        exit 0
    fi
}

_build_base() {
    _verify_build_base
    set -e
    _prepare_netrc
    
    echo "[info] -- build and push docker base image $DOCKER_IMAGE:$IMAGE_TAG_BASE"
    docker build -t $DOCKER_IMAGE:$IMAGE_TAG_BASE . -f $DOCKERFILE_BASE
    docker push $DOCKER_IMAGE:$IMAGE_TAG_BASE
}

_prepare_netrc() {
    echo "[info] -- Prepare .netrc"
    echo "$NETRC" > "build/ci/.netrc"
}

_build() {
    _validation
    set -e

    #set docker image tag
    APP_NAME=$(basename $CMD_PATH | tr _ -)

    # if implementing platform worker services
    if [[ "$CMD_PATH" == *"worker"* ]]; then
        BASE=$(basename $CMD_PATH | tr _ -)
        APP_NAME="worker-${BASE}"
    fi
    
    IMAGE_VERSION=$(git rev-parse --short HEAD)-$(date +%y%m%d%H%M%S)
    IMAGE_TAG=$APP_NAME.$ENV_NAME-$IMAGE_VERSION

    echo "[info] -- building docker image for $DOCKER_IMAGE:$IMAGE_TAG"
    docker build -t $DOCKER_IMAGE:$IMAGE_TAG -f $DOCKERFILE . \
        --build-arg DOCKER_IMAGE=$DOCKER_IMAGE \
        --build-arg BASE_TAG=$IMAGE_TAG_BASE \
        --build-arg SERVICE_PATH=$CMD_PATH

    case "$PUSH" in
    no-push)
        echo "[info] -- push image disabled"
        ;;
    *)
        echo "[info] -- pushing $DOCKER_IMAGE:$IMAGE_TAG"
        docker push $DOCKER_IMAGE:$IMAGE_TAG
        ;;
    esac
}

# _setup_git_cred() {
#     set -e
#     echo "[info] -- setup git credential"
#     go env -w GOPRIVATE=github.com/urbanindo/*
#     mkdir -p /root/.ssh
#     cp config/.ssh/id_rsa /root/.ssh/id_rsa
#     chmod 400 /root/.ssh/id_rsa
#     ssh-keyscan -t rsa github.com > /root/.ssh/known_hosts
#     printf "Host github.com\n\tStrictHostKeyChecking no\n\tIdentityFile ~/.ssh/id_rsa" > /root/.ssh/config
#     git config --global url.git@github.com:.insteadOf https://github.com/
#     ssh -T git@github.com || echo ""
#     set +e
# }

_init_test() {
    echo "[info] -- initialize test"
    _prepare_netrc
    mv build/ci/.netrc $HOME/.netrc
    go env -w GOPRIVATE=$GOPRIVATE
}

_run_test() {
    # _setup_git_cred
    echo "[info] -- running unit test"
    set -e
    make run-test
}

_unshallow_git() {
    # _setup_git_cred
    git fetch --unshallow || git fetch --all
}

_run_lint() {
    echo "[info] - run lint"
    make vetting
}

_init() {
    echo "[info] -- load .env config"
    if [[ -n $CI_CONFIG_DIR ]]; then
        CI_CONFIG=$CI_CONFIG_DIR
    else
        CI_CONFIG=ci.env
    fi
    export $(grep -v '^#' $CI_CONFIG | xargs)
}

_usage() {
    echo "Usage : CI_STAGE=build $0"
}

_version() {
    echo "Version : 1.1.0"
}

_main "$@"; exit