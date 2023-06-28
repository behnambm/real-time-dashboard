build_data_source:
	@go build -o ./datasource/datasource.out ./datasource/main.go

build_server:
	@go build -o ./service/server.out ./service/main.go


compose: build_data_source build_server
	@docker compose up --build


