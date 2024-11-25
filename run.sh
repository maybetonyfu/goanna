#!/bin/bash

fastapi run web.py --port 8090 &

./goanna &

wait -n

exit $?