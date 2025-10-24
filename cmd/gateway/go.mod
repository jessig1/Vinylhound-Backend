module vinylhound/api-gateway

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	vinylhound/shared v0.0.0
)

replace vinylhound/shared => ../../shared/go
