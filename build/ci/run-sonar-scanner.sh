#!/usr/bin/env bash

if [[ $CI = "true" ]] && [[ $TRAVIS = "true" ]]; then
    if [[ $TRAVIS_PULL_REQUEST = "false" ]]; then
        echo "sonar.branch.name=$TRAVIS_BRANCH" >> sonar-project.properties
    else
        echo "sonar.pullrequest.key=$TRAVIS_PULL_REQUEST" >> sonar-project.properties
        echo "sonar.pullrequest.branch=$TRAVIS_PULL_REQUEST_BRANCH" >> sonar-project.properties
        echo "sonar.pullrequest.base=$TRAVIS_BRANCH" >> sonar-project.properties
    fi
fi

echo "######## sonar-project.properties ########"
cat sonar-project.properties
echo "##########################################"

bash <(curl -s -L https://gist.githubusercontent.com/setyolegowo/5a6a9aaf9693d8645357f016de02b1fc/raw/33def3df34c1412a82da3b8d8b70b72b8e88ab0b/run-sonar-scanner.sh)