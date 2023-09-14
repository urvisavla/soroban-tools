package test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"github.com/google/shlex"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/icmd"
)

func TestCLIContractInstall(t *testing.T) {
	NewCLITest(t)
	output := runSuccessfulCLICmd(t, "contract install --wasm "+helloWorldContractPath)
	wasm := getHelloWorldContract(t)
	contractHash := xdr.Hash(sha256.Sum256(wasm))
	require.Contains(t, output, contractHash.HexString())
}

func TestCLIContractInstallAndDeploy(t *testing.T) {
	NewCLITest(t)
	runSuccessfulCLICmd(t, "contract install --wasm "+helloWorldContractPath)
	wasm := getHelloWorldContract(t)
	contractHash := xdr.Hash(sha256.Sum256(wasm))
	output := runSuccessfulCLICmd(t, fmt.Sprintf("contract deploy --salt 0 --wasm-hash %s", contractHash.HexString()))
	outputsContractIDInLastLine(t, output)
}

func outputsContractIDInLastLine(t *testing.T, output string) {
	lines := strings.Split(output, "\n")
	nonEmptyLines := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			nonEmptyLines = append(nonEmptyLines, l)
		}
	}
	require.GreaterOrEqual(t, len(nonEmptyLines), 1)
	contractID := nonEmptyLines[len(nonEmptyLines)-1]
	require.Len(t, contractID, 56)
	require.Regexp(t, "^C", contractID)
}

func TestCLIContractDeploy(t *testing.T) {
	NewCLITest(t)
	output := runSuccessfulCLICmd(t, "contract deploy --salt 0 --wasm "+helloWorldContractPath)
	outputsContractIDInLastLine(t, output)
}

func TestCLIContractDeployAndInvoke(t *testing.T) {
	NewCLITest(t)
	output := runSuccessfulCLICmd(t, "contract deploy --salt=0 --wasm "+helloWorldContractPath)
	contractID := strings.TrimSpace(output)
	output = runSuccessfulCLICmd(t, fmt.Sprintf("contract invoke --id %s -- hello --world=world", contractID))
	require.Contains(t, output, `["Hello","world"]`)
}

func runSuccessfulCLICmd(t *testing.T, cmd string) string {
	res := runCLICommand(t, cmd)
	require.NoError(t, res.Error, fmt.Sprintf("stderr:\n%s\nstdout:\n%s\n", res.Stderr(), res.Stdout()))
	return res.Stdout()
}

func runCLICommand(t *testing.T, cmd string) *icmd.Result {
	args := []string{"run", "-q", "--", "--vv"}
	parsedArgs, err := shlex.Split(cmd)
	require.NoError(t, err, cmd)
	args = append(args, parsedArgs...)
	c := icmd.Command("cargo", args...)
	c.Env = append(os.Environ(),
		fmt.Sprintf("SOROBAN_RPC_URL=http://localhost:%d/", sorobanRPCPort),
		fmt.Sprintf("SOROBAN_NETWORK_PASSPHRASE=%s", StandaloneNetworkPassphrase),
	)
	return icmd.RunCmd(c)
}

func NewCLITest(t *testing.T) *Test {
	test := NewTest(t)
	ch := jhttp.NewChannel(test.sorobanRPCURL(), nil)
	client := jrpc2.NewClient(ch, nil)

	sourceAccount := keypair.Root(StandaloneNetworkPassphrase)

	// Create default account used by the CLI
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: keypair.Root(StandaloneNetworkPassphrase).Address(),
			Sequence:  1,
		},
		IncrementSequenceNum: false,
		Operations: []txnbuild.Operation{&txnbuild.CreateAccount{
			Destination: "GDIY6AQQ75WMD4W46EYB7O6UYMHOCGQHLAQGQTKHDX4J2DYQCHVCR4W4",
			Amount:      "100000",
		}},
		BaseFee: txnbuild.MinBaseFee,
		Memo:    nil,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	require.NoError(t, err)
	sendSuccessfulTransaction(t, client, sourceAccount, tx)
	return test
}