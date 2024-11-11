#!/bin/bash

./goanna &

fastapi run web.py --port 8090 &

wait -n

exit $?