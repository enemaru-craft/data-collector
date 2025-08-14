FUNCTION_NAME := stg_power_data_registration_lambda
ZIP_FILE := lambda_function.zip
BINARY_NAME := bootstrap
SOURCE_FILE := main.go
AWS_PAGER := ""

.PHONY: build zip deploy clean all

all: build zip deploy

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY_NAME) $(SOURCE_FILE)

zip: build
	zip $(ZIP_FILE) $(BINARY_NAME)

deploy: zip
	AWS_PAGER="" aws lambda update-function-code \
		--region ap-northeast-1 \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(ZIP_FILE)

clean:
	rm -f $(BINARY_NAME) $(ZIP_FILE)
