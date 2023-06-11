// Package rpcfuzz is meant to have some basic RPC fuzzing and
// conformance tests
package rpcfuzz

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/maticnetwork/polygon-cli/rpctypes"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
	"os"
	"regexp"
	"strings"
)

type (
	// RPCTest is the common interface for a test
	RPCTest interface {
		// GetMethod returns the json rpc method name
		GetMethod() string

		// GetArgs will return the list of arguments that will be used when calling the rpc
		GetArgs() []interface{}

		// Validate will return an error of the result fails validation
		Validate(result interface{}) error

		// ExpectError is used by the validation code to understand of the test typically returns an error
		ExpectError() bool
	}

	// RPCTestGenric is the simplist implementation of the
	// RPCTest. Basically the implementation of the interface is
	// managed by just returning hard coded values for method,
	// args, validator, and error
	RPCTestGeneric struct {
		Method    string
		Args      []interface{}
		Validator func(result interface{}) error
		IsError   bool
	}
)

const (
	codeQualityPrivateKey = "42b6e34dc21598a807dc19d7784c71b2a7a01f6480dc6f58258f78e539f1a1fa"
)

var (
	testPrivateHexKey   *string
	testContractAddress *string
	testPrivateKey      *ecdsa.PrivateKey
	testEthAddress      ethcommon.Address
)

var (
	RPCTestNetVersion                       RPCTestGeneric
	RPCTestWeb3ClientVersion                RPCTestGeneric
	RPCTestWeb3SHA3                         RPCTestGeneric
	RPCTestWeb3SHA3Error                    RPCTestGeneric
	RPCTestNetListening                     RPCTestGeneric
	RPCTestNetPeerCount                     RPCTestGeneric
	RPCTestEthProtocolVersion               RPCTestGeneric
	RPCTestEthSyncing                       RPCTestGeneric
	RPCTestEthCoinbase                      RPCTestGeneric
	RPCTestEthChainID                       RPCTestGeneric
	RPCTestEthMining                        RPCTestGeneric
	RPCTestEthHashrate                      RPCTestGeneric
	RPCTestEthGasPrice                      RPCTestGeneric
	RPCTestEthAccounts                      RPCTestGeneric
	RPCTestEthBlockNumber                   RPCTestGeneric
	RPCTestEthGetBalanceLatest              RPCTestGeneric
	RPCTestEthGetBalanceEarliest            RPCTestGeneric
	RPCTestEthGetBalancePending             RPCTestGeneric
	RPCTestEthGetStorageAtLatest            RPCTestGeneric
	RPCTestEthGetStorageAtEarliest          RPCTestGeneric
	RPCTestEthGetStorageAtPending           RPCTestGeneric
	RPCTestEthGetTransactionCountAtLatest   RPCTestGeneric
	RPCTestEthGetTransactionCountAtEarliest RPCTestGeneric
	RPCTestEthGetTransactionCountAtPending  RPCTestGeneric

	RPCTestEthBlockByNumber RPCTestGeneric

	allTests = make([]RPCTest, 0)
)

func setupTests() {
	// cast rpc --rpc-url localhost:8545 net_version
	RPCTestNetVersion = RPCTestGeneric{
		Method:    "net_version",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^\d*$`),
	}
	allTests = append(allTests, &RPCTestNetVersion)

	// cast rpc --rpc-url localhost:8545 web3_clientVersion
	RPCTestWeb3ClientVersion = RPCTestGeneric{
		Method:    "web3_clientVersion",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^[[:print:]]*$`),
	}
	allTests = append(allTests, &RPCTestWeb3ClientVersion)

	// cast rpc --rpc-url localhost:8545 web3_sha3 0x68656c6c6f20776f726c64
	RPCTestWeb3SHA3 = RPCTestGeneric{
		Method:    "web3_sha3",
		Args:      []interface{}{"0x68656c6c6f20776f726c64"},
		Validator: ValidateRegexString(`0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad`),
	}
	allTests = append(allTests, &RPCTestWeb3SHA3)

	RPCTestWeb3SHA3Error = RPCTestGeneric{
		IsError:   true,
		Method:    "web3_sha3",
		Args:      []interface{}{"68656c6c6f20776f726c64"},
		Validator: ValidateError(`cannot unmarshal hex string without 0x prefix`),
	}
	allTests = append(allTests, &RPCTestWeb3SHA3Error)

	// cast rpc --rpc-url localhost:8545 net_listening
	RPCTestNetListening = RPCTestGeneric{
		Method:    "net_listening",
		Args:      []interface{}{},
		Validator: ValidateExact(true),
	}
	allTests = append(allTests, &RPCTestNetListening)

	// cast rpc --rpc-url localhost:8545 net_peerCount
	RPCTestNetPeerCount = RPCTestGeneric{
		Method:    "net_peerCount",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]*$`),
	}
	allTests = append(allTests, &RPCTestNetPeerCount)

	// cast rpc --rpc-url localhost:8545 eth_protocolVersion
	RPCTestEthProtocolVersion = RPCTestGeneric{
		IsError:   true,
		Method:    "eth_protocolVersion",
		Args:      []interface{}{},
		Validator: ValidateError(`method eth_protocolVersion does not exist`),
	}
	allTests = append(allTests, &RPCTestEthProtocolVersion)

	// cast rpc --rpc-url localhost:8545 eth_syncing
	RPCTestEthSyncing = RPCTestGeneric{
		Method: "eth_syncing",
		Args:   []interface{}{},
		Validator: ChainValidator(
			ValidateExact(false),
			ValidateJSONSchema(rpctypes.RPCSchemaEthSyncing),
		),
	}
	allTests = append(allTests, &RPCTestEthSyncing)

	// cast rpc --rpc-url localhost:8545 eth_coinbase
	RPCTestEthCoinbase = RPCTestGeneric{
		Method:    "eth_coinbase",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{40}$`),
	}
	allTests = append(allTests, &RPCTestEthCoinbase)

	// cast rpc --rpc-url localhost:8545 eth_chainId
	RPCTestEthChainID = RPCTestGeneric{
		Method:    "eth_chainId",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthChainID)

	// cast rpc --rpc-url localhost:8545 eth_mining
	RPCTestEthMining = RPCTestGeneric{
		Method: "eth_mining",
		Args:   []interface{}{},
		Validator: ChainValidator(
			ValidateExact(true),
			ValidateExact(false),
		),
	}
	allTests = append(allTests, &RPCTestEthMining)

	// cast rpc --rpc-url localhost:8545 eth_hashrate
	RPCTestEthHashrate = RPCTestGeneric{
		Method:    "eth_hashrate",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthHashrate)

	// cast rpc --rpc-url localhost:8545 eth_gasPrice
	RPCTestEthGasPrice = RPCTestGeneric{
		Method:    "eth_gasPrice",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGasPrice)

	// cast rpc --rpc-url localhost:8545 eth_accounts
	RPCTestEthAccounts = RPCTestGeneric{
		Method:    "eth_accounts",
		Args:      []interface{}{},
		Validator: ValidateJSONSchema(rpctypes.RPCSchemaAccountList),
	}
	allTests = append(allTests, &RPCTestEthAccounts)

	// cast rpc --rpc-url localhost:8545 eth_blockNumber
	RPCTestEthBlockNumber = RPCTestGeneric{
		Method:    "eth_blockNumber",
		Args:      []interface{}{},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthBlockNumber)

	// cast balance --rpc-url localhost:8545 0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6
	RPCTestEthGetBalanceLatest = RPCTestGeneric{
		Method:    "eth_getBalance",
		Args:      []interface{}{testEthAddress.String(), "latest"},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGetBalanceLatest)
	RPCTestEthGetBalanceEarliest = RPCTestGeneric{
		Method:    "eth_getBalance",
		Args:      []interface{}{testEthAddress.String(), "earliest"},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGetBalanceEarliest)
	RPCTestEthGetBalancePending = RPCTestGeneric{
		Method:    "eth_getBalance",
		Args:      []interface{}{testEthAddress.String(), "pending"},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGetBalancePending)

	// cast storage --rpc-url localhost:8545 0x6fda56c57b0acadb96ed5624ac500c0429d59429 3
	RPCTestEthGetStorageAtLatest = RPCTestGeneric{
		Method:    "eth_getStorageAt",
		Args:      []interface{}{*testContractAddress, "0x3", "latest"},
		Validator: ValidateRegexString(`^0x000000000000000000000000` + strings.ToLower(testEthAddress.String())[2:] + `$`),
	}
	allTests = append(allTests, &RPCTestEthGetStorageAtLatest)
	RPCTestEthGetStorageAtEarliest = RPCTestGeneric{
		Method:    "eth_getStorageAt",
		Args:      []interface{}{*testContractAddress, "0x3", "earliest"},
		Validator: ValidateRegexString(`^0x0{64}`),
	}
	allTests = append(allTests, &RPCTestEthGetStorageAtEarliest)
	RPCTestEthGetStorageAtPending = RPCTestGeneric{
		Method:    "eth_getStorageAt",
		Args:      []interface{}{*testContractAddress, "0x3", "pending"},
		Validator: ValidateRegexString(`^0x000000000000000000000000` + strings.ToLower(testEthAddress.String())[2:] + `$`),
	}
	allTests = append(allTests, &RPCTestEthGetStorageAtPending)

	// cast rpc --rpc-url localhost:8545 eth_getTransactionCount 0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6 latest
	RPCTestEthGetTransactionCountAtLatest = RPCTestGeneric{
		Method:    "eth_getTransactionCount",
		Args:      []interface{}{testEthAddress.String(), "latest"},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGetTransactionCountAtLatest)
	RPCTestEthGetTransactionCountAtEarliest = RPCTestGeneric{
		Method:    "eth_getTransactionCount",
		Args:      []interface{}{testEthAddress.String(), "earliest"},
		Validator: ValidateRegexString(`^0x0$`),
	}
	allTests = append(allTests, &RPCTestEthGetTransactionCountAtEarliest)
	RPCTestEthGetTransactionCountAtPending = RPCTestGeneric{
		Method:    "eth_getTransactionCount",
		Args:      []interface{}{testEthAddress.String(), "pending"},
		Validator: ValidateRegexString(`^0x[[:xdigit:]]{1,}$`),
	}
	allTests = append(allTests, &RPCTestEthGetTransactionCountAtPending)

	// spacing this thing out
	// spacing this thing out
	// spacing this thing out
	// spacing this thing out
	// spacing this thing out
	// spacing this thing out
	// cast block --rpc-url localhost:8545 0
	RPCTestEthBlockByNumber = RPCTestGeneric{
		Method:    "eth_getBlockByNumber",
		Args:      []interface{}{"0x0", true},
		Validator: ValidateJSONSchema(rpctypes.RPCSchemaEthBlock),
	}
	allTests = append(allTests, &RPCTestEthBlockByNumber)

}

// ChainValidator would take a list of validation functions to be
// applied in order. The idea is that if first validator is true, then
// the rest won't be applied.
func ChainValidator(validators ...func(interface{}) error) func(result interface{}) error {
	return func(result interface{}) error {
		for _, v := range validators {
			err := v(result)
			if err == nil {
				return nil
			}
		}
		return fmt.Errorf("All Validation failed")
	}

}

// ValidateJSONSchema is used to validate the response against a JSON Schema
func ValidateJSONSchema(schema string) func(result interface{}) error {
	return func(result interface{}) error {
		validatorLoader := gojsonschema.NewStringLoader(schema)

		// This is weird, but the current setup doesn't allow
		// for easy access to the initial response string...
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("Unable to marshal result back to json for validation: %w", err)
		}
		responseLoader := gojsonschema.NewStringLoader(string(jsonBytes))

		validatorResult, err := gojsonschema.Validate(validatorLoader, responseLoader)
		if err != nil {
			return fmt.Errorf("Unable to run json validation: %w", err)
		}
		// fmt.Println(string(jsonBytes))
		if !validatorResult.Valid() {
			errStr := ""
			for _, desc := range validatorResult.Errors() {
				errStr += desc.String() + "\n"
			}
			return fmt.Errorf("The json document is not valid: %s", errStr)
		}
		return nil

	}
}

// ValidateExact will validate against the exact value expected.
func ValidateExact(expected interface{}) func(result interface{}) error {
	return func(result interface{}) error {
		if expected != result {
			return fmt.Errorf("Expected %v and got %v", expected, result)
		}
		return nil
	}
}

// ValidateRegexString will match a string from the json response against a regular expression
func ValidateRegexString(regEx string) func(result interface{}) error {
	r := regexp.MustCompile(regEx)
	return func(result interface{}) error {
		resultStr, isValid := result.(string)
		if !isValid {
			return fmt.Errorf("Invalid result type. Expected string but got %T", result)
		}
		if !r.MatchString(resultStr) {
			return fmt.Errorf("The regex %s failed to match result %s", regEx, resultStr)
		}
		return nil
	}
}

// ValidateError will check the error message text against the provide regular expression
func ValidateError(errorMessageRegex string) func(result interface{}) error {
	r := regexp.MustCompile(errorMessageRegex)
	return func(result interface{}) error {
		resultError, isValid := result.(error)
		if !isValid {
			return fmt.Errorf("Invalid result type. Expected error but got %T", result)
		}
		if !r.MatchString(resultError.Error()) {
			return fmt.Errorf("The regex %s failed to match result %s", errorMessageRegex, resultError.Error())
		}
		return nil
	}
}

func (r *RPCTestGeneric) GetMethod() string {
	return r.Method
}
func (r *RPCTestGeneric) GetArgs() []interface{} {
	return r.Args
}
func (r *RPCTestGeneric) Validate(result interface{}) error {
	return r.Validator(result)
}
func (r *RPCTestGeneric) ExpectError() bool {
	return r.IsError
}

var RPCFuzzCmd = &cobra.Command{
	Use:   "rpcfuzz http://localhost:8545",
	Short: "Continually run a variety of RPC calls and fuzzers",
	Long: `

This command will run a series of RPC calls against a given json rpc
endpoint. The idea is to be able to check for various features and
function to see if the RPC generally conforms to typical geth
standards for the RPC

Some setup might be neede depending on how you're testing. We'll
demonstrate with geth. In order to quickly test this, you can run geth
in dev mode:

# ./build/bin/geth --dev --dev.period 5 --http --http.addr localhost \
    --http.port 8545 \
    --http.api admin,debug,web3,eth,txpool,personal,miner,net \
    --verbosity 5 --rpc.gascap 50000000  --rpc.txfeecap 0 \
    --miner.gaslimit  10 --miner.gasprice 1 --gpo.blocks 1 \
    --gpo.percentile 1 --gpo.maxprice 10 --gpo.ignoreprice 2 \
    --dev.gaslimit 50000000

Once your Eth client is running and the RPC is functional, you'll need
to transfer some amount of ether to a known account that ca be used
for testing

# cast send --from "$(cast rpc --rpc-url localhost:8545 eth_coinbase | jq -r '.')" \
    --rpc-url localhost:8545 --unlocked --value 100ether \
    0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6

Then we might want to deploy some test smart contracts. For the
purposes of testing we'll use Uniswap v3 at this hash
d8b1c635c275d2a9450bd6a78f3fa2484fef73eb

# cast send --from 0x85dA99c8a7C2C95964c8EfD687E95E632Fc533D6 \
    --private-key 0x42b6e34dc21598a807dc19d7784c71b2a7a01f6480dc6f58258f78e539f1a1fa \
    --rpc-url localhost:8545 --create \
    "$(jq -r '.bytecode' ~/code/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json)"

Once this has been completed this will be the address of the contract:
0x6fda56c57b0acadb96ed5624ac500c0429d59429

- https://ethereum.github.io/execution-apis/api-documentation/
- https://ethereum.org/en/developers/docs/apis/json-rpc/
- https://json-schema.org/
- https://www.liquid-technologies.com/online-json-to-schema-converter

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cxt := cmd.Context()
		rpcClient, err := rpc.DialContext(cxt, args[0])
		if err != nil {
			return err
		}
		log.Trace().Msg("Doing test setup")
		setupTests()

		for _, t := range allTests {
			log.Trace().Str("method", t.GetMethod()).Msg("Running Test")
			var result interface{}
			err = rpcClient.CallContext(cxt, &result, t.GetMethod(), t.GetArgs()...)
			if err != nil && !t.ExpectError() {
				log.Error().Err(err).Str("method", t.GetMethod()).Msg("Method test failed")
				continue
			}

			if t.ExpectError() {
				err = t.Validate(err)
			} else {
				err = t.Validate(result)
			}

			if err != nil {
				log.Error().Err(err).Str("method", t.GetMethod()).Msg("Failed to validate")
				continue
			}
			log.Info().Str("method", t.GetMethod()).Msg("Successfully validated")
		}
		return nil
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Expected 1 argument, but got %d", len(args))
		}

		privateKey, err := ethcrypto.HexToECDSA(*testPrivateHexKey)
		if err != nil {
			log.Error().Err(err).Msg("Couldn't process the hex private key")
			return err
		}

		ethAddress := ethcrypto.PubkeyToAddress(privateKey.PublicKey)
		log.Info().Str("ethAddress", ethAddress.String()).Msg("Loaded private key")

		testPrivateKey = privateKey
		testEthAddress = ethAddress

		return nil
	},
}

func init() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flagSet := RPCFuzzCmd.PersistentFlags()
	testPrivateHexKey = flagSet.String("private-key", codeQualityPrivateKey, "The hex encoded private key that we'll use to sending transactions")
	testContractAddress = flagSet.String("contract-address", "0x6fda56c57b0acadb96ed5624ac500c0429d59429", "The address of a contract that can be used for testing")

}
