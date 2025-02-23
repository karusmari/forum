#!/bin/bash
docker build -t forum .
docker run -d --name forum_app -p 8080:8080 forum