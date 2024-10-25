#!/bin/bash

./goanna &

fastapi run web.py &

wait -n

exit $?