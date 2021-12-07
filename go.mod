module ovpn-health-check

go 1.17

require (
	github.com/aws/aws-sdk-go v1.42.20
	github.com/gorilla/mux v1.8.0
	github.com/mmattice/go-openvpn-mgmt v0.0.0-20211126224253-86f82af9eebd
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/sclasen/go-metrics-cloudwatch v0.0.0-20180222121429-246549584841
)

require github.com/jmespath/go-jmespath v0.4.0 // indirect
