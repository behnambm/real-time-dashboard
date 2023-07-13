build_data_source:
	@go build -o ./datasource/datasource.out ./datasource/*.go

build_server:
	@go build -o ./service/server.out ./service/*.go


up: build_data_source build_server
	@docker compose up --build


