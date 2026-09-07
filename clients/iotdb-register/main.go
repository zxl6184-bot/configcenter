package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git/pkg/configcenter"
	"dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git/pkg/module"
)

func main() {
	var (
		serviceName = flag.String("name", "iotdb-client1", "service name registered in config center")
		routerName  = flag.String("router", "API", "router name of this client service")
		address     = flag.String("address", "http://127.0.0.1:8080", "router address of this client service")
		timeout     = flag.Duration("timeout", 60*time.Second, "timeout to wait for the iotdb instance")
	)
	flag.Parse()

	// The SDK reads CONFIG_CENTER_ADDRESS to locate the config center.
	os.Setenv("CONFIG_CENTER_ADDRESS", "172.21.92.5:32405")
	ccAddr := os.Getenv("CONFIG_CENTER_ADDRESS")
	if ccAddr == "" {
		fmt.Fprintln(os.Stderr, "environment variable CONFIG_CENTER_ADDRESS is not set")
		os.Exit(1)
	}

	moduleCfg := module.ModuleConfig{
		Name: *serviceName,
		ServiceInstances: []*configcenter.ServiceInstance{
			{
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_IoTDB,
			}, {
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_Postgres,
			}, {
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_Mongodb,
			}, {
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_RabbitMQ,
			}, {
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_IoTDB,
			}, {
				// apply for an IoTDB instance, the config center
				// creates an independent database and user for this service
				ServiceInstanceType: configcenter.ServiceInstance_Redis,
			},
		},
		Routers: []*configcenter.Router{
			{
				Name:    *routerName,
				Address: *address,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := module.NewModule(moduleCfg, ctx)
	go m.LinkConfigCenter()

	fmt.Printf("connect to config center: %s, register service: %s\n", ccAddr, *serviceName)

	deadline := time.Now().Add(*timeout)
	for time.Now().Before(deadline) {
		dsn := m.GetServiceInstancesUnsafe()
		if len(dsn) > 0 {
			fmt.Println("======== iotdb service instance ========")
			fmt.Println(dsn)
			fmt.Println("========================================")
			fmt.Printf("status: %s\n", m.GetStatus().String())
			return
		}

		fmt.Printf("[%s] waiting for iotdb instance, status=%s\n",
			time.Now().Format("15:04:05"), m.GetStatus().String())
		time.Sleep(2 * time.Second)
	}

	fmt.Fprintln(os.Stderr, "timeout waiting for iotdb service instance")
	os.Exit(1)
}
