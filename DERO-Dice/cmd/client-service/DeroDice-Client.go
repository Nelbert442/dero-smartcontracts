package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/graviton"
	"github.com/docopt/docopt-go"
	"github.com/ybbus/jsonrpc"
)

type GravitonStore struct {
	DB            *graviton.Store
	DBFolder      string
	DBPath        string
	DBTree        string
	migrating     int
	DBMaxSnapshot uint64
	DBMigrateWait time.Duration
	Writing       int
}

type TreeKV struct {
	k []byte
	v []byte
}

type TXDetails struct {
	multiplier uint64
	function   string
	userKey    string
	txid       string
	amount     uint64
}

// Mainnet TODO: Adding params for default vals like website ssl, default multiplier, default function, etc.
var command_line string = `DeroDice-Client
DERO Dice Service (client): A 2 way service implementation for interacting with a Smart Contract while ensuring input variables are as private as possible

Usage:
  DeroDice-Client [options]
  DeroDice-Client -h | --help

Options:
  -h --help     Show this screen.
  --rpc-server-address=<127.0.0.1:40403>	connect to service (client) wallet
  --scid=<73535cbe3254d0943d52e4b2b94dcf98d29868b364c21046c57bb8575218474f>	define SCID that is leveraged (NOTE: MUST BE SAME ON CLIENT AND SERVER)`

// Some constant vars, in future Mainnet TODO: implementation these will be properly defined in config/other .go integrations
const PLUGIN_NAME = "Dero_Dice"

const DEFAULT_MULTIPLIER = uint64(2)
const DEFAULT_FUNCTION = "RollDiceHigh"

const DEST_PORT = uint64(0x8765432187654321)

var walletRPCClient jsonrpc.RPCClient
var derodice_scid string

var Graviton_backend *GravitonStore = &GravitonStore{}

// Main function that provisions persistent graviton store, gets listening wallet addr & service listeners spun up and calls looped function to keep service alive
func main() {
	var err error
	var walletEndpoint string

	var arguments map[string]interface{}

	if err != nil {
		log.Fatalf("Error while parsing arguments err: %s\n", err)
	}

	arguments, err = docopt.Parse(command_line, nil, true, "DERO Dice Client : work in progress", false)
	_ = arguments

	log.Printf("DERO Dice Service (client) :  This is under heavy development, use it for testing/evaluations purpose only\n")

	// Set variables from arguments
	walletEndpoint = "127.0.0.1:40403"
	if arguments["--rpc-server-address"] != nil {
		walletEndpoint = arguments["--rpc-server-address"].(string)
	}

	log.Printf("Using wallet RPC endpoint %s\n", walletEndpoint)

	derodice_scid = "73535cbe3254d0943d52e4b2b94dcf98d29868b364c21046c57bb8575218474f"
	if arguments["--scid"] != nil {
		derodice_scid = arguments["--scid"].(string)
	}

	log.Printf("Using SCID %s", derodice_scid)

	// create wallet client
	walletRPCClient = jsonrpc.NewClient("http://" + walletEndpoint + "/json_rpc")

	// Test rpc-server connection to ensure wallet connectivity, exit out if not
	var addr *rpc.Address
	var addr_result rpc.GetAddress_Result
	err = walletRPCClient.CallFor(&addr_result, "GetAddress")
	if err != nil || addr_result.Address == "" {
		log.Printf("Could not obtain address from wallet (http://%s/json_rpc) err %s\n", walletEndpoint, err)
		return
	}

	if addr, err = rpc.NewAddress(addr_result.Address); err != nil {
		log.Printf("address could not be parsed: addr:%s err:%s\n", addr_result.Address, err)
		return
	}

	shasum := fmt.Sprintf("%x", sha1.Sum([]byte(addr.String())))

	db_folder := fmt.Sprintf("%s_%s", PLUGIN_NAME, shasum)

	Graviton_backend.NewGravDB("derodice", db_folder, "25ms", 5000)

	log.Printf("Persistant store for processed txids created in '%s'\n", db_folder)

	processing_thread() // rkeep processing
}

// Keep processing open for service
func processing_thread() {
	var err error

	for { // currently we traverse entire history

		time.Sleep(time.Second)

		var transfers rpc.Get_Transfers_Result
		err = walletRPCClient.CallFor(&transfers, "GetTransfers", rpc.Get_Transfers_Params{In: true, SourcePort: DEST_PORT})
		if err != nil {
			log.Printf("Could not obtain gettransfers from wallet err %s\n", err)
			continue
		}

		for _, e := range transfers.Entries {
			// Sets the default values in the event "Comment" isn't supplied or validated against - future vals will be in a config json/.go format
			multiplier := DEFAULT_MULTIPLIER
			scFunction := DEFAULT_FUNCTION
			var userKey string

			if e.Coinbase || !e.Incoming { // skip coinbase or outgoing, self generated transactions
				continue
			}

			// check whether the entry has been processed before, if yes skip it
			var already_processed bool

			// Get txDetail [sender+txid] from graviton store, if received it is already processed else continue
			txDetails := Graviton_backend.GetTX(e.TXID)
			if txDetails != nil {
				already_processed = true
			}

			if already_processed { // if already processed skip it
				continue
			}

			// Mainnet TODO: Make the function names dynamic of sorts.. or perhaps trust the service will "fix" all and some other comparison can be leveraged
			if e.Payload_RPC.Has(rpc.RPC_COMMENT, rpc.DataString) {
				payloadComment := e.Payload_RPC.Value(rpc.RPC_COMMENT, rpc.DataString).(string)

				multFunc := strings.Split(payloadComment, "/")
				if len(multFunc) > 1 {
					multiplier, _ = strconv.ParseUint(multFunc[0], 10, 64)

					if multFunc[1] == "RollDiceHigh" || multFunc[1] == "RollDiceLow" {
						scFunction = multFunc[1]
					}

					userKey = multFunc[2]
				}
			} else {
				continue
			}

			if userKey == "" {
				log.Printf("err generating userKey. Using temp 'tempKey' to keep processes moving")
				userKey = "tempKey"
			}

			// check whether this service should handle the transfer
			if !e.Payload_RPC.Has(rpc.RPC_SOURCE_PORT, rpc.DataUint64) ||
				DEST_PORT != e.Payload_RPC.Value(rpc.RPC_SOURCE_PORT, rpc.DataUint64).(uint64) { // this service is expecting value to be specfic
				log.Printf("err with service handle transfer check - look into it. Source_Port: %v, Destination_Port: %v", rpc.RPC_SOURCE_PORT, rpc.RPC_DESTINATION_PORT)
				continue
			}

			log.Printf("tx should be processed %s\n", e.TXID)

			// Store new txdetails in graviton store
			// TODO: Set multiplier to a known value that is sent from user, otherwise default is 2 atm
			newTxDetails := TXDetails{multiplier: multiplier, function: scFunction, userKey: userKey, txid: e.TXID, amount: e.Amount}
			err = Graviton_backend.StoreTX(newTxDetails)

			if err != nil {
				log.Printf("err updating db to err %s\n", err)
			} else {
				log.Printf("[Processed Successfully] TX Reply sent")
			}

			// Sleep inbetween tx generations - helps fix unknown errs atm: TODO
			log.Printf("Sleeping 5s for safety...")
			time.Sleep(5 * time.Second)

			// Send received detail to DeroDice SC!
			log.Printf("Now time to send data to SC for DeroDice gameplay!")

			// Build smart contract TX
			var scstr string
			var rpcArgs []rpc.Argument
			rpcArgs = append(rpcArgs, rpc.Argument{Name: "entrypoint", DataType: "S", Value: scFunction})
			rpcArgs = append(rpcArgs, rpc.Argument{Name: "multiplier", DataType: "U", Value: multiplier})
			rpcArgs = append(rpcArgs, rpc.Argument{Name: "userKey", DataType: "S", Value: userKey})
			scparams := rpc.SC_Invoke_Params{SC_ID: derodice_scid, SC_DERO_Deposit: e.Amount, SC_RPC: rpcArgs}
			err = walletRPCClient.CallFor(&scstr, "scinvoke", scparams)
			if err != nil {
				log.Printf("sending SC tx err %s\n", err)
				continue
			}
			log.Printf("[Processed Successfully] SC data sent. SCID: %v, Amount: %v, Function: %v, Multiplier: %v, UserKey: %v", derodice_scid, e.Amount, scFunction, multiplier, userKey)

			// Sleep inbetween tx generations - helps fix unknown errs atm: TODO
			log.Printf("Sleeping 60s to process next incoming tx for safety...")
			time.Sleep(60 * time.Second)
		}
	}
}

// ---- Graviton/Backend functions ---- //
// Mainnet TODO: Proper graviton/backend .go file(s)
// Builds new Graviton DB based on input from main()
func (g *GravitonStore) NewGravDB(poolhost, dbFolder, dbmigratewait string, dbmaxsnapshot uint64) {
	current_path, err := os.Getwd()
	if err != nil {
		log.Printf("%v", err)
	}

	g.DBMigrateWait, _ = time.ParseDuration(dbmigratewait)

	g.DBMaxSnapshot = dbmaxsnapshot

	g.DBFolder = dbFolder

	g.DBPath = filepath.Join(current_path, dbFolder)

	g.DB, err = graviton.NewDiskStore(g.DBPath)
	if err != nil {
		log.Fatalf("Could not create db store: %v", err)
	}

	g.DBTree = poolhost
}

// Swaps the store pointer from existing to new after copying latest snapshot to new DB - fast as cursor + disk writes allow [possible other alternatives such as mem store for some of these interwoven, testing needed]
func (g *GravitonStore) SwapGravDB(poolhost, dbFolder string) {
	// Use g.migrating as a simple 'mutex' of sorts to lock other read/write functions out of doing anything with DB until this function has completed.
	g.migrating = 1

	// Rename existing bak to bak2, then goroutine to cleanup so process doesn't wait for old db cleanup time
	var bakFolder string = dbFolder + "_bak"
	var bak2Folder string = dbFolder + "_bak2"
	log.Printf("Renaming directory %v to %v", bakFolder, bak2Folder)
	os.Rename(bakFolder, bak2Folder)
	log.Printf("Removing directory %v", bak2Folder)
	go os.RemoveAll(bak2Folder)

	// Get existing store values, defer close of original, and get store values for new DB to write to
	store := g.DB
	ss, _ := store.LoadSnapshot(0)

	tree, _ := ss.GetTree(g.DBTree)
	log.Printf("SS: %v", ss.GetVersion())

	c := tree.Cursor()
	log.Printf("Getting k/v pairs")
	// Duplicate the LATEST (snapshot 0) to the new DB, this starts the DB over again, but still retaining X number of old DBs for version in future use cases. Here we get the vals before swapping to new db in mem
	var treeKV []*TreeKV // Just k & v which are of type []byte
	for k, v, err := c.First(); err == nil; k, v, err = c.Next() {
		temp := &TreeKV{k, v}
		treeKV = append(treeKV, temp)
	}
	log.Printf("Closing store")
	store.Close()

	// Backup last set of g.DBMaxSnapshot snapshots, can offload elsewhere or make this process as X many times as you want to backup.
	var oldFolder string
	oldFolder = g.DBPath
	log.Printf("Renaming directory %v to %v", oldFolder, bakFolder)
	os.Rename(oldFolder, bakFolder)

	log.Printf("Creating new disk store")
	g.DB, _ = graviton.NewDiskStore(g.DBPath)

	// Take vals from previous DB store that were put into treeKV struct (array of), and commit to new DB after putting all k/v pairs back
	store = g.DB
	ss, _ = store.LoadSnapshot(0)
	tree, _ = ss.GetTree(g.DBTree)

	log.Printf("Putting k/v pairs into tree...")
	for _, val := range treeKV {
		tree.Put(val.k, val.v)
	}
	log.Printf("Committing k/v pairs to tree")
	_, cerr := graviton.Commit(tree)
	if cerr != nil {
		log.Printf("[Graviton] ERROR: %v", cerr)
	}
	log.Printf("Migration to new DB is done.")
	g.migrating = 0
}

// Gets TX details
func (g *GravitonStore) GetTX(txid string) *TXDetails {
	store := g.DB
	ss, _ := store.LoadSnapshot(0) // load most recent snapshot

	// Swap DB at g.DBMaxSnapshot+ commits. Check for g.migrating, if so sleep for g.DBMigrateWait ms
	for g.migrating == 1 {
		log.Printf("[GetTX] G is migrating... sleeping for %v...", g.DBMigrateWait)
		time.Sleep(g.DBMigrateWait)
		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}
	if ss.GetVersion() >= g.DBMaxSnapshot {
		Graviton_backend.SwapGravDB(Graviton_backend.DBTree, Graviton_backend.DBFolder)

		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}

	tree, _ := ss.GetTree(g.DBTree) // use or create tree named by poolhost in config
	key := txid
	var reply *TXDetails

	//log.Printf("Getting key: %v", key)
	v, _ := tree.Get([]byte(key))
	if v != nil {
		//log.Printf("Key found...")
		_ = json.Unmarshal(v, &reply)
		return reply
	}

	return nil
}

// Stores TX details
func (g *GravitonStore) StoreTX(txDetails TXDetails) error {
	confBytes, err := json.Marshal(txDetails)
	if err != nil {
		return fmt.Errorf("[Graviton] could not marshal txDetails info: %v", err)
	}

	store := g.DB
	ss, _ := store.LoadSnapshot(0) // load most recent snapshot

	// Swap DB at g.DBMaxSnapshot+ commits. Check for g.migrating, if so sleep for g.DBMigrateWait ms
	for g.migrating == 1 {
		log.Printf("[StoreTX] G is migrating... sleeping for %v...", g.DBMigrateWait)
		time.Sleep(g.DBMigrateWait)
		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}
	if ss.GetVersion() >= g.DBMaxSnapshot {
		Graviton_backend.SwapGravDB(Graviton_backend.DBTree, Graviton_backend.DBFolder)

		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}

	tree, _ := ss.GetTree(g.DBTree) // use or create tree named by poolhost in config
	key := txDetails.txid
	log.Printf("[Graviton-StoreTX] Storing %v", txDetails)
	tree.Put([]byte(key), []byte(confBytes)) // insert a value
	_, cerr := graviton.Commit(tree)
	if cerr != nil {
		log.Printf("[Graviton] ERROR: %v", cerr)
	}
	return nil
}

// ---- End Graviton/Backend functions ---- //
