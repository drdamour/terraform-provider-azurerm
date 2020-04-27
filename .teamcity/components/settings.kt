// specifies the default hour (UTC) at which tests should be triggered, if enabled
var defaultStartHour = 4

// specifies the default level of parallelism per-service-package
var defaultParallelism = 10

var locations = mapOf(
        "public" to LocationConfiguration("westeurope", "eastus2", "francecentral", false),
        "germany" to LocationConfiguration("westeurope", "eastus2", "francecentral", false)
)

// specifies the list of Azure Environments where tests should be run nightly
var runNightly = mapOf(
        "public" to true
)

// specifies a list of services which should be run with a custom test configuration
var serviceTestConfigurationOverrides = mapOf(
        // The AKS API has a low rate limit
        "containers" to testConfiguration(5, 5),
        "compute" to testConfiguration(5, 4),

        // Data Lake has a low quota
        "datalake" to testConfiguration(2, 5)
)