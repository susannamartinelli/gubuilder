## GuBuilder
This utility helps you to build, test, build-image and upload a new microservice
image on AWS ECR your repository.

from root dir
- Unix OS
```
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo geouniq.com/gubuilder
```

- Mac OS
```
CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo geouniq.com/gubuilder
```

Then
```
go install -ldflags "-s -X main.Version=`git describe --always --tags`" geouniq.com/gubuilder
```

You should see `gubuilder` command into your path (if not reopen terminal or type `rehash` command)

## Prerequisites
 - `.ssh` in your `$HOME`
 - `.aws/credentials` in your `$HOME`
 - you must run in a dir with  `glide.yaml` and `Dockerfile` files

## Usage
 - `git clone git@gitlab.com:yourdomain/your-prj.git`
 - cd directory `your-prj`
 - type `gubuilder [command] [options]` 
 - help for usage `gubuilder -h` 


