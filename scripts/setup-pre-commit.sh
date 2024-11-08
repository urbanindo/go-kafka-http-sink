#!/bin/bash

if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit could not be found, head to https://pre-commit.com/#install to install pre-commit on your machine"
    exit
fi

echo "installing pre-commit hooks"
pre-commit install

echo "Done. pre-commit is ready to use on this repository"
