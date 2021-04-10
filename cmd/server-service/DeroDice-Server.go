package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/graviton"
	"github.com/docopt/docopt-go"
	"github.com/gorilla/mux"
	"github.com/ybbus/jsonrpc"
)

type GetInfoReply struct {
	Difficulty         int64   `json:"difficulty"`
	Stableheight       int64   `json:"stableheight"`
	Topoheight         int64   `json:"topoheight"`
	Averageblocktime50 float32 `json:"averageblocktime50"`
	Target             int64   `json:"target"`
	Testnet            bool    `json:"testnet"`
	TopBlockHash       string  `json:"top_block_hash"`
	DynamicFeePerKB    int64   `json:"dynamic_fee_per_kb"`
	TotalSupply        int64   `json:"total_supply"`
	MedianBlockSize    int64   `json:"median_block_Size"`
	Version            string  `json:"version"`
	Height             int64   `json:"height"`
	TxPoolSize         int64   `json:"tx_pool_size"`
	Status             string  `json:"status"`
}

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
	DestinationAddress string
	Multiplier         uint64
	Function           string
	UserKey            string
	Txid               string
	Txrpc              string
	Amount             uint64
}

type RollResults struct {
	UserKey    string
	Won        bool
	Multiplier uint64
	TopoHeight uint64
	TxID       string
}

type DiceRolls struct {
	RollTXs []*TXDetails
}

type DiceRollResults struct {
	RollResults []*RollResults
}

type RollOdds struct {
	Multiplier      uint64
	BaseRatio       string
	HistoricalRatio string
}

type ApiServer struct {
	stats            atomic.Value
	statsIntv        string
	customBetAddress string
	setBetAddress    string
}

type Website struct {
	Enabled  bool
	Port     string
	SSL      bool
	SSLPort  string
	CertFile string
	KeyFile  string
}

var command_line string = `DeroDice-Server
DERO Dice Service (server): A 2 way service implementation for interacting with a Smart Contract while ensuring input variables are as private as possible

Usage:
  DeroDice-Server [options]
  DeroDice-Server -h | --help

Options:
  -h --help     Show this screen.
  --daemon-address=<host:port>	Use daemon instance at <host>:<port> or https://domain
  --rpc-server-address=<127.0.0.1:40403>	connect to service (server) wallet
  --scid=<73535cbe3254d0943d52e4b2b94dcf98d29868b364c21046c57bb8575218474f>	define SCID that is leveraged (NOTE: MUST BE SAME ON CLIENT AND SERVER)
  --api-address=<0.0.0.0:8224>	API (non-SSL) will be enabled at the defined address (or defaulted to 0.0.0.0:8224)
  --ssl-api-address=<0.0.0.0:8225>	if defined, API (SSL) will be enabled at the defined address. apifullchain.cer && apicert.key in the same dir is required
  --frontend-port=<8080>	if defined, frontend (non-SSL) will be enabled
  --ssl-frontend-port=<8181>	if defined, frontend (SSL) will be enabled. fefullchain.cer && fecert.key in the same dir is required`

// Some constant vars, in future Mainnet TODO: implementation these will be properly defined in config/other .go integrations
var api_nonssl_addr string
var api_ssl_addr string
var api_use_ssl bool

const API_CERTFILE = "apifullchain.cer"
const API_KEYFILE = "apicert.key"

const PLUGIN_NAME = "Dero_Dice"
const USER_KEY_SIZE = 8
const MAX_VALUE_EXPECTED = 1000000 // Mainnet TODO: Will grab this value from the SC itself (maxwager)
const MIN_VALUE_EXPECTED = 50000   // Mainnet TODO: Will grab this value from the SC itself (minwager)
const DEFAULT_MULTIPLIER = uint64(2)
const DEFAULT_FUNCTION = "RollDiceHigh"

const DEST_PORT = uint64(0x8765432187654321)

var derodice_scid string

// TODO: Update min and max bet dynamically from SC values (have to perform a call from a daemon initially, check if local is running else use a remote)
var expected_arguments = rpc.Arguments{
	{rpc.RPC_DESTINATION_PORT, rpc.DataUint64, DEST_PORT},
	{rpc.RPC_COMMENT, rpc.DataString, fmt.Sprintf("DERO Dice! Min Bet: %v , Max Bet: %v", walletapi.FormatMoney(MIN_VALUE_EXPECTED), walletapi.FormatMoney(MAX_VALUE_EXPECTED))},
	{"Comment", rpc.DataString, fmt.Sprintf("Multiplier/Function (If parse err, default will be %v/%v):", DEFAULT_MULTIPLIER, DEFAULT_FUNCTION)},
	{rpc.RPC_VALUE_TRANSFER, rpc.DataUint64, uint64(100000)}, // in atomic units

}

// currently the interpreter seems to have a glitch if this gets initialized within the code
// see limitations github.com/traefik/yaegi
var response = rpc.Arguments{
	{rpc.RPC_DESTINATION_PORT, rpc.DataUint64, uint64(0)},
	{rpc.RPC_SOURCE_PORT, rpc.DataUint64, DEST_PORT},
	{rpc.RPC_COMMENT, rpc.DataString, "DERO Dice!"},
}

// jsonRPC Clients
var walletRPCClient jsonrpc.RPCClient
var derodRPCClient jsonrpc.RPCClient

// Top-level declarations, Mainnet TODO: Add to config of sorts later
var Graviton_backend *GravitonStore = &GravitonStore{}
var API *ApiServer = &ApiServer{
	statsIntv: "10s",
}

// Main function that provisions persistent graviton store, gets listening wallet addr & service listeners spun up and calls looped function to keep service alive
func main() {
	var err error
	var walletEndpoint, derodEndpoint string

	var arguments map[string]interface{}

	if err != nil {
		log.Fatalf("Error while parsing arguments err: %s\n", err)
	}

	arguments, err = docopt.Parse(command_line, nil, true, "DERO Dice Server : work in progress", false)
	_ = arguments

	log.Printf("DERO Dice Service (server) :  This is under heavy development, use it for testing/evaluations purpose only\n")

	// Set variables from arguments
	walletEndpoint = "127.0.0.1:40403"
	if arguments["--rpc-server-address"] != nil {
		walletEndpoint = arguments["--rpc-server-address"].(string)
	}

	log.Printf("Using wallet RPC endpoint %s", walletEndpoint)

	derodEndpoint = "127.0.0.1:40402"
	if arguments["--daemon-address"] != nil {
		derodEndpoint = arguments["--daemon-address"].(string)
	}

	log.Printf("Using derod RPC endpoint %s", derodEndpoint)

	derodice_scid = "73535cbe3254d0943d52e4b2b94dcf98d29868b364c21046c57bb8575218474f"
	if arguments["--scid"] != nil {
		derodice_scid = arguments["--scid"].(string)
	}

	log.Printf("Using SCID %s", derodice_scid)

	// create wallet client
	walletRPCClient = jsonrpc.NewClient("http://" + walletEndpoint + "/json_rpc")

	// create derod client
	derodRPCClient = jsonrpc.NewClient("http://" + derodEndpoint + "/json_rpc")

	api_use_ssl = false
	api_nonssl_addr = "0.0.0.0:8224"
	if arguments["--api-address"] != nil {
		api_nonssl_addr = arguments["--api-address"].(string)
	}

	api_ssl_addr = "0.0.0.0:8225"
	if arguments["--ssl-api-address"] != nil {
		api_use_ssl = true
	}

	var frontend_port, ssl_frontend_port string
	var frontend_ssl_enabled, frontend_enabled bool

	if arguments["--frontend-port"] != nil {
		frontend_port = arguments["--frontend-port"].(string)
		frontend_enabled = true
	}

	if arguments["--ssl-frontend-port"] != nil {
		ssl_frontend_port = arguments["--ssl-frontend-port"].(string)
		frontend_ssl_enabled = true
		frontend_enabled = true
	}

	// Define website params
	var web *Website = &Website{
		Enabled:  frontend_enabled,
		Port:     frontend_port,
		SSL:      frontend_ssl_enabled,
		SSLPort:  ssl_frontend_port,
		CertFile: "fefullchain.cer",
		KeyFile:  "fecert.key",
	}

	// Test rpc to derod
	var str *GetInfoReply
	err = derodRPCClient.CallFor(&str, "get_info")
	if err != nil {
		log.Printf("getting lastblock header err %s\n", err)
		return
	}

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

	log.Printf("Persistant store created in '%s'\n", db_folder)

	log.Printf("Wallet Address: %s\n", addr)
	service_address_without_amount := addr.Clone()
	service_address_without_amount.Arguments = expected_arguments[:len(expected_arguments)-1]
	log.Printf("Integrated address to activate '%s', (without hardcoded amount) service: \n%s\n", PLUGIN_NAME, service_address_without_amount.String())
	API.customBetAddress = service_address_without_amount.String()

	// service address can be created client side for now
	service_address := addr.Clone()
	service_address.Arguments = expected_arguments
	log.Printf("Integrated address to activate '%s', service: \n%s\n", PLUGIN_NAME, service_address.String())
	API.setBetAddress = service_address.String()

	go api_process(API) // start api process / listener
	if web.Enabled {
		go web_process(web) // start web process / listener
	}

	processing_thread() // rkeep processing
}

// Keep website running
func web_process(cfg *Website) {
	fileServer := http.FileServer(http.Dir("./site"))
	http.Handle("/", fileServer)

	// If SSL is enabled, configure for SSL and HTTP. Else just run HTTP
	if cfg.SSL {
		go func() {
			log.Printf("[Website] Starting website at port %v\n", cfg.Port)
			addr := ":" + cfg.Port
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				log.Printf("[Website] Error starting http server at %v", addr)
				log.Fatal(err)
			}
		}()

		log.Printf("[Website] Starting SSL website at port %v\n", cfg.SSLPort)

		addr := ":" + cfg.SSLPort
		err := http.ListenAndServeTLS(addr, cfg.CertFile, cfg.KeyFile, nil)
		if err != nil {
			log.Printf("[Website] Error starting https server at %v", addr)
			log.Fatal(err)
		}
	} else {
		log.Printf("[Website] Starting website at port %v\n", cfg.Port)

		addr := ":" + cfg.Port
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Printf("[Website] Error starting http server at %v", addr)
			log.Fatal(err)
		}
	}
}

// Keep api running
func api_process(cfg *ApiServer) {
	statsIntv, _ := time.ParseDuration(cfg.statsIntv)
	statsTimer := time.NewTimer(statsIntv)
	log.Printf("[API] Set stats collect interval to %v", statsIntv)

	collectStats()

	go func() {
		for {
			select {
			case <-statsTimer.C:
				collectStats()
				statsTimer.Reset(statsIntv)
			}
		}
	}()

	// If SSL is configured, due to nature of listenandserve, put HTTP in go routine then call SSL afterwards so they can run in parallel. Otherwise, run http as normal
	if api_use_ssl {
		go apiListen()
		apiListenSSL()
	} else {
		apiListen()
	}
}

// Keep processing open for service
func processing_thread() {
	var err error

	for { // currently we traverse entire history

		time.Sleep(time.Second)

		var transfers rpc.Get_Transfers_Result
		err = walletRPCClient.CallFor(&transfers, "GetTransfers", rpc.Get_Transfers_Params{In: true, DestinationPort: DEST_PORT})
		if err != nil {
			log.Printf("Could not obtain gettransfers from wallet err %s\n", err)
			continue
		}

		for _, e := range transfers.Entries {
			// Sets the default values in the event "Comment" isn't supplied or validated against - future vals will be in a config json/.go format
			multiplier := DEFAULT_MULTIPLIER
			scFunction := DEFAULT_FUNCTION

			if e.Coinbase || !e.Incoming { // skip coinbase or outgoing, self generated transactions
				continue
			}

			// Mainnet TODO: Make the function names dynamic of sorts, pull from SC itself and reference them. Also allow for lowercase vs camel case scenarios instead of requiring exact match
			if e.Payload_RPC.Has("Comment", rpc.DataString) {
				payloadComment := e.Payload_RPC.Value("Comment", rpc.DataString).(string)

				multFunc := strings.Split(payloadComment, "/")
				if len(multFunc) > 1 {
					multiplier, _ = strconv.ParseUint(multFunc[0], 10, 64)

					if multFunc[1] == "RollDiceHigh" || multFunc[1] == "RollDiceLow" {
						scFunction = multFunc[1]
					}
				}
			}

			// check whether the entry has been processed before, if yes skip it
			var already_processed bool

			// Get txDetail [sender+txid] from graviton store, if received it is already processed else continue
			txDetails := Graviton_backend.GetTXs()

			// Loop through TXs to see if txid exists
			for _, v := range txDetails {
				if txDetails != nil {
					if v != nil {
						if v.DestinationAddress == e.Sender && v.Txid == e.TXID {
							already_processed = true
						}
					}
				}
			}

			if already_processed { // if already processed skip it
				continue
			}

			// check whether this service should handle the transfer
			if !e.Payload_RPC.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) ||
				DEST_PORT != e.Payload_RPC.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64) { // this service is expecting value to be specfic
				continue

			}

			log.Printf("tx should be processed %s\n", e.TXID)

			if expected_arguments.Has(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) { // this service is expecting value to be within a range

				// Gen random userKey , if exists will continue and loop back through to be processed later. // Mainet TODO: do/while or some other check & balance here instead to ensure continuity
				var randKey [USER_KEY_SIZE]byte
				userKeyByte := randKey[0 : USER_KEY_SIZE-1]
				rand.Read(randKey[:])
				copy(randKey[:], userKeyByte[:])

				userKey := hex.EncodeToString(userKeyByte)

				// If userKey exists, continue so that it'll get picked back up
				allTxs := Graviton_backend.GetTXs()

				// Loop through TXs to see if userKey exists
				var userKeyExists bool
				for _, v := range allTxs {
					if allTxs != nil {
						if v != nil {
							if v.UserKey == userKey {
								userKeyExists = true
							}
						}
					}
				}
				if userKeyExists {
					log.Printf("UserKey (%v) already exists, continuing. Will be processed later.", userKey)
					continue
				}

				response[0].Value = e.SourcePort // source port now becomes destination port, similar to TCP
				txReply := fmt.Sprintf("%v/%s/%s", multiplier, scFunction, userKey)
				response[2].Value = txReply

				// sender of ping now becomes destination
				var str string
				//tparams := rpc.Transfer_Params{Transfers: []rpc.Transfer{{Destination: e.Sender, Amount: uint64(1), Payload_RPC: response}}}
				tparams := rpc.Transfer_Params{Transfers: []rpc.Transfer{{Destination: e.Sender, Amount: e.Amount, Payload_RPC: response}}}
				err = walletRPCClient.CallFor(&str, "Transfer", tparams)
				if err != nil {
					log.Printf("sending reply tx err %s\n", err)
					continue
				}

				// Store new txdetails in graviton store
				newTxDetails := &TXDetails{DestinationAddress: e.Sender, Multiplier: multiplier, Function: scFunction, UserKey: userKey, Txid: e.TXID, Txrpc: txReply, Amount: e.Amount}
				err = Graviton_backend.StoreTX(newTxDetails)

				if err != nil {
					log.Printf("err updating db to err %s\n", err)
				} else {
					log.Printf("[Processed Successfully] TX Reply sent")
				}

				// Mainnet TODO: Sleep inbetween tx generations - helps fix unknown errs atm
				log.Printf("Sleeping 60s to process next incoming tx for safety...")
				time.Sleep(60 * time.Second)
			}
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
func (g *GravitonStore) GetTXs() []*TXDetails {
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
	key := "dicerolls"
	var reply *DiceRolls

	//log.Printf("Getting key: %v", key)
	v, _ := tree.Get([]byte(key))
	if v != nil {
		//log.Printf("Key found...")
		_ = json.Unmarshal(v, &reply)
		return reply.RollTXs
	}

	return nil
}

// Stores TX details
func (g *GravitonStore) StoreTX(txDetails *TXDetails) error {
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

	tree, _ := ss.GetTree(g.DBTree)
	key := "dicerolls"

	currDiceRolls, err := tree.Get([]byte(key))
	var diceRolls *DiceRolls

	var newDiceRoll []byte

	if err != nil {
		// Returns key not found if != nil, or other err, but assuming keynotfound/leafnotfound
		var txDetailsArr []*TXDetails
		txDetailsArr = append(txDetailsArr, txDetails)
		diceRolls = &DiceRolls{RollTXs: txDetailsArr}
	} else {
		// Retrieve value and convert to BlocksFoundByHeight, so that you can manipulate and update db
		_ = json.Unmarshal(currDiceRolls, &diceRolls)

		diceRolls.RollTXs = append(diceRolls.RollTXs, txDetails)
	}
	newDiceRoll, err = json.Marshal(diceRolls)
	if err != nil {
		return fmt.Errorf("[Graviton] could not marshal diceRolls info: %v", err)
	}

	log.Printf("[Graviton-StoreTX] Storing %v", txDetails)
	tree.Put([]byte(key), []byte(newDiceRoll)) // insert a value
	_, cerr := graviton.Commit(tree)
	if cerr != nil {
		log.Printf("[Graviton] ERROR: %v", cerr)
	}
	return nil
}

// Gets dice roll result details
func (g *GravitonStore) GetRollResults() []*RollResults {
	store := g.DB
	ss, _ := store.LoadSnapshot(0) // load most recent snapshot

	// Swap DB at g.DBMaxSnapshot+ commits. Check for g.migrating, if so sleep for g.DBMigrateWait ms
	for g.migrating == 1 {
		log.Printf("[GetRollResults] G is migrating... sleeping for %v...", g.DBMigrateWait)
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
	key := "rollresults"
	var reply *DiceRollResults

	//log.Printf("Getting key: %v", key)
	v, _ := tree.Get([]byte(key))
	if v != nil {
		//log.Printf("Key found...")
		_ = json.Unmarshal(v, &reply)
		return reply.RollResults
	}

	return nil
}

// Stores dice roll results
func (g *GravitonStore) StoreRollResults(results *RollResults) error {
	store := g.DB
	ss, _ := store.LoadSnapshot(0) // load most recent snapshot

	// Swap DB at g.DBMaxSnapshot+ commits. Check for g.migrating, if so sleep for g.DBMigrateWait ms
	for g.migrating == 1 {
		log.Printf("[StoreRollResults] G is migrating... sleeping for %v...", g.DBMigrateWait)
		time.Sleep(g.DBMigrateWait)
		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}
	if ss.GetVersion() >= g.DBMaxSnapshot {
		Graviton_backend.SwapGravDB(Graviton_backend.DBTree, Graviton_backend.DBFolder)

		store = g.DB
		ss, _ = store.LoadSnapshot(0) // load most recent snapshot
	}

	tree, _ := ss.GetTree(g.DBTree)
	key := "rollresults"

	currRollResults, err := tree.Get([]byte(key))
	var rollResults *DiceRollResults

	var newRollResults []byte

	if err != nil {
		// Returns key not found if != nil, or other err, but assuming keynotfound/leafnotfound
		var rollResultsArr []*RollResults
		rollResultsArr = append(rollResultsArr, results)
		rollResults = &DiceRollResults{RollResults: rollResultsArr}
	} else {
		// Retrieve value and convert to BlocksFoundByHeight, so that you can manipulate and update db
		_ = json.Unmarshal(currRollResults, &rollResults)

		rollResults.RollResults = append(rollResults.RollResults, results)
	}
	newRollResults, err = json.Marshal(rollResults)
	if err != nil {
		return fmt.Errorf("[Graviton] could not marshal rollResults info: %v", err)
	}

	log.Printf("[Graviton-StoreTX] Storing %v", results)
	tree.Put([]byte(key), []byte(newRollResults)) // insert a value
	_, cerr := graviton.Commit(tree)
	if cerr != nil {
		log.Printf("[Graviton] ERROR: %v", cerr)
	}
	return nil
}

// ---- End Graviton/Backend functions ---- //

// ---- API functions ---- //
// Mainnet TODO: Proper api .go file(s)
// API Server listen over non-SSL
func apiListen() {
	log.Printf("[API] Starting API on %v", api_nonssl_addr)
	router := mux.NewRouter()
	router.HandleFunc("/api/stats", statsIndex)
	router.NotFoundHandler = http.HandlerFunc(notFound)
	err := http.ListenAndServe(api_nonssl_addr, router)
	if err != nil {
		log.Fatalf("[API] Failed to start API: %v", err)
	}
}

// API Server listen over SSL
func apiListenSSL() {
	log.Printf("[API] Starting SSL API on %v", api_ssl_addr)
	routerSSL := mux.NewRouter()
	routerSSL.HandleFunc("/api/stats", statsIndex)
	routerSSL.NotFoundHandler = http.HandlerFunc(notFound)
	err := http.ListenAndServeTLS(api_ssl_addr, API_CERTFILE, API_KEYFILE, routerSSL)
	if err != nil {
		log.Fatalf("[API] Failed to start SSL API: %v", err)
	}
}

// Serve the notfound addr
func notFound(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.WriteHeader(http.StatusNotFound)
}

// API Collect Stats
func collectStats() {
	stats := make(map[string]interface{})

	allTxs := Graviton_backend.GetTXs()
	allRollResults := Graviton_backend.GetRollResults()

	// Loop through TXs to see if txid exists
	// Mainnet TODO: Most likely more efficient way(s) to perform these actions, but for dARCH Event 0 will be this - optimizations later
	for _, v := range allTxs {
		if allTxs != nil {
			if v != nil {
				var resultsStored bool

				// Check if roll results exist for current userKey
				for _, r := range allRollResults {
					if allRollResults != nil {
						if r != nil {
							if r.UserKey == v.UserKey {
								resultsStored = true
							}
						}
					}
				}

				if !resultsStored {
					// Store new txdetails in graviton store
					userKeyResults := checkUserKeyResults(v.UserKey)
					var txid string
					var multiplier, topoHeight uint64
					var won bool

					// Parse userKeyResults
					// TODO: Split by : (0 is win/lose/err), rest is details to define
					multRes := strings.Split(userKeyResults, ":")
					if len(multRes) > 1 {
						wonStr := multRes[0]

						if wonStr == "Lose" {
							won = false

							resParse := strings.Split(multRes[1], ",")
							if len(resParse) > 1 {
								multiplier, _ = strconv.ParseUint(resParse[0], 10, 64)
								topoHeight, _ = strconv.ParseUint(resParse[1], 10, 64)
								txid = resParse[2]
							} else {
								log.Printf("Err with %v results definition: %v", wonStr, multRes[1])
							}

							newRollResults := &RollResults{UserKey: v.UserKey, Won: won, Multiplier: multiplier, TopoHeight: topoHeight, TxID: txid}
							Graviton_backend.StoreRollResults(newRollResults)
						} else if wonStr == "Win" {
							won = true

							resParse := strings.Split(multRes[1], ",")
							if len(resParse) > 1 {
								multiplier, _ = strconv.ParseUint(resParse[0], 10, 64)
								topoHeight, _ = strconv.ParseUint(resParse[1], 10, 64)
								txid = resParse[2]
							} else {
								log.Printf("Err with %v results definition: %v", wonStr, multRes[1])
							}

							newRollResults := &RollResults{UserKey: v.UserKey, Won: won, Multiplier: multiplier, TopoHeight: topoHeight, TxID: txid}
							Graviton_backend.StoreRollResults(newRollResults)
						} else if wonStr == "Err" {
							log.Printf("Err result for %v", v.UserKey)
						} else {
							// You will get to this point if checking on a userKey that hasn't propogated over to the SC yet. Let it roll through this else, log if you want to for debug
							// There's really no way to know when the client interacts with SC until the var is available from SC. It's possible it can err forever, just let it roll.
							log.Printf("Waiting for SC keystore for '%v' (this can take time).", v.UserKey) // Mainnet TODO: Perhaps time estimation by looking to ensure that tx has left server side
						}
					} else {
						log.Printf("Shouldn't be here, incorrect results returned: %v", userKeyResults)
					}
				} else {
					//log.Printf("Results already stored for %v, continuing.", v.UserKey)
				}
			}
		}
	}

	apiRollResults := Graviton_backend.GetRollResults()

	rollResults, wCount, lCount, totalPlayed := convertRollResults(apiRollResults)

	// Define rollResults [*RollResults] alongside win and loss counts by multiplier
	stats["rollResults"] = rollResults
	stats["totalPlayed"] = totalPlayed
	stats["coin"] = "DERO"
	stats["transactionExplorer"] = "https://testnetexplorer.dero.io/tx/{id}"
	stats["customBetAddress"] = API.customBetAddress
	stats["setBetAddress"] = API.setBetAddress

	// Define Odds
	var DiceRollOdds []*RollOdds

	for i := uint64(2); i <= uint64(10); i++ {
		currRollOdd := &RollOdds{}
		currRollOdd.Multiplier = i
		currRollOdd.BaseRatio = fmt.Sprintf("%v / 100", 100/i)
		currRollOdd.HistoricalRatio = fmt.Sprintf("%v / %v", wCount[i], wCount[i]+lCount[i])
		DiceRollOdds = append(DiceRollOdds, currRollOdd)
	}

	stats["rollOdds"] = DiceRollOdds

	API.stats.Store(stats)
}

// Converts and returns necessary values to feed into API from GetRollResults()
func convertRollResults(rollResults []*RollResults) ([]*RollResults, map[uint64]int64, map[uint64]int64, int64) {
	//wlRatio := make(map[uint64]*WLRatio)
	wCount := make(map[uint64]int64)
	lCount := make(map[uint64]int64)
	var totalPlayed int64

	for _, currRollResults := range rollResults {
		if rollResults != nil {
			if currRollResults != nil {
				if currRollResults.Won == true {
					if wCount[currRollResults.Multiplier] > 0 {
						wCount[currRollResults.Multiplier] += 1
					} else {
						wCount[currRollResults.Multiplier] = 1
					}
					totalPlayed += 1
				} else if currRollResults.Won == false {
					if lCount[currRollResults.Multiplier] > 0 {
						lCount[currRollResults.Multiplier] += 1
					} else {
						lCount[currRollResults.Multiplier] = 1
					}
					totalPlayed += 1
				}
			}
		}
	}

	return rollResults, wCount, lCount, totalPlayed
}

// API StatsIndex
func statsIndex(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.WriteHeader(http.StatusOK)

	reply := make(map[string]interface{})

	stats := getStats()
	if stats != nil {
		reply["rollResults"] = stats["rollResults"]
		reply["totalPlayed"] = stats["totalPlayed"]
		reply["coin"] = stats["coin"]
		reply["transactionExplorer"] = stats["transactionExplorer"]
		reply["customBetAddress"] = stats["customBetAddress"]
		reply["setBetAddress"] = stats["setBetAddress"]
		reply["rollOdds"] = stats["rollOdds"]
	}

	err := json.NewEncoder(writer).Encode(reply)
	if err != nil {
		log.Printf("[API] Error serializing API response: %v", err)
	}
}

// API Get stats from backend
func getStats() map[string]interface{} {
	stats := API.stats.Load()
	if stats != nil {
		return stats.(map[string]interface{})
	}
	return nil
}

// ---- End API functions ---- //

// ---- SC variable check functions ---- //
func checkUserKeyResults(userKey string) string {
	// Grab userKey value from SC
	var scstr *rpc.GetSC_Result
	var strings []string
	strings = append(strings, userKey)
	getSC := rpc.GetSC_Params{SCID: derodice_scid, Code: false, KeysString: strings}
	err := derodRPCClient.CallFor(&scstr, "getsc", getSC)
	if err != nil {
		log.Printf("getting SC tx err %s\n", err)
		return ""
	}

	if len(scstr.ValuesString) > 1 {
		log.Printf("more than 1 value returned for '%v'. Will be returning slot 0, here are all of them: %v\n", userKey, scstr.ValuesString)
	}

	return scstr.ValuesString[0]
}

// ---- End SC variable check functions ---- //
