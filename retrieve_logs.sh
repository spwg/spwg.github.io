#!/bin/bash

rsync -a -v -z -e "ssh -i ~/.ssh/spencerwgreene" spencergreene@104.154.159.64:~/spencerwgreene.com/http/logs/ .
