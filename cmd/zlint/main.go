package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/zmap/zcrypto/x509"
	"github.com/zmap/zlint"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var ( //flags
	inPath           string
	outPath          string
	numCertThreads   int
	prettyPrint      bool
	numProcs         int
	channelSize      int
	crashIfParseFail bool
	outProcessPath   string
)

var fileMutex sync.Mutex

func init() {
	flag.StringVar(&inPath, "input-file", "", "File path for the input certificate(s).")
	flag.StringVar(&outPath, "output-file", "-", "File path for the output JSON.")
	flag.StringVar(&outProcessPath, "output-process", "-", "File path for output error preprocess.")
	flag.BoolVar(&prettyPrint, "list-lints-json", false, "Use this flag to print supported lints in JSON format, one per line")
	flag.IntVar(&numCertThreads, "cert-threads", 1, "Use this flag to specify the number of threads in -threads mode.  This has no effect otherwise.")
	flag.IntVar(&numProcs, "procs", 0, "Use this flag to specify the number of processes to run on.")
	flag.IntVar(&channelSize, "channel-size", 1000000, "Use this flag to specify the number of values in the buffered channel.")
	flag.BoolVar(&crashIfParseFail, "fatal-parse-errors", false, "Fatally crash if a certificate cannot be parsed. Log by default.")
	flag.Parse()
}

func CustomMarshal(validation interface{}, lintResult *zlint.ResultSet, raw []byte, parsed *x509.Certificate) ([]byte, error) {
	return json.Marshal(struct {
		Raw        []byte            `json:"raw,omitempty"`
		Parsed     *x509.Certificate `json:"parsed,omitempty"`
		ZLint      *zlint.ResultSet  `json:"zlint,omitempty"`
		Validation interface{}       `json:"validation,omitempty"`
	}{
		Raw:        raw,
		Parsed:     parsed,
		ZLint:      lintResult,
		Validation: validation,
	})
}

type Validation struct {
	nssValid    bool
	nssWasValid bool
}

func MakeIssuerString(cert *x509.Certificate, result *zlint.ResultSet, validationInterface interface{}) string {
	validation := FillOutValidationStruct(validationInterface)
	issuerDn := cert.Issuer.String()
	issuerDn = strings.Replace(issuerDn, ", ", ":", -1)
	subjectDn := cert.Subject.String()
	subjectDn = strings.Replace(subjectDn, ", ", ":", -1)
	numErrors := len(result.Errors)
	numWarnings := len(result.Warnings)
	numFatals := len(result.Fatals)
	notBefore := cert.NotBefore.String()
	notAfter := cert.NotAfter.String()
	sha256fp := cert.FingerprintSHA256.Hex()

	var outputString string
	outputString += strconv.Itoa(numErrors) + "," + strconv.Itoa(numWarnings) + "," + strconv.FormatBool(validation.nssValid) + "," + strconv.FormatBool(validation.nssWasValid) + "," + notBefore + "," + notAfter + "," + issuerDn + "," + subjectDn + "," + sha256fp + "," + strconv.Itoa(numFatals) + "," + strings.Join(result.Errors, ",") + "," + strings.Join(result.Warnings, ",") + "\n"
	return outputString
}

func FillOutValidationStruct(validation interface{}) *Validation {
	v := Validation{}
	validationMap := validation.(map[string]interface{})
	nssMap := validationMap["nss"].(map[string]interface{})

	v.nssValid = nssMap["valid"].(bool)
	v.nssWasValid = nssMap["was_valid"].(bool)

	return &v
}

func ProcessCertificate(in <-chan interface{}, out chan<- []byte, outFile *os.File, outProcessFile *os.File, wg *sync.WaitGroup) {
	log.Info("Processing certificates...")
	defer wg.Done()
	for raw := range in {
		zdbData := raw.(map[string]interface{})
		raw := zdbData["raw"]
		validation := zdbData["validation"]
		der, err := base64.StdEncoding.DecodeString(raw.(string))
		parsed, err := x509.ParseCertificate(der)
		if err != nil { //could not parse
			if crashIfParseFail {
				log.Fatal("could not parse certificate with error: ", err)
			} else {
				log.Info("could not parse certificate with error: ", err)
			}
		} else { //parsed
			zlintResult := zlint.LintCertificate(parsed)
			var processedString string
			if validation != nil {
				processedString = MakeIssuerString(parsed, zlintResult, validation)
			} else {
				processedString = "\n"
			}
			jsonResult, err := CustomMarshal(validation, zlintResult, der, parsed)
			if err != nil {
				log.Fatal("could not parse JSON.")
			}
			fileMutex.Lock()
			outProcessFile.WriteString(processedString)
			outFile.Write(jsonResult)
			outFile.Write([]byte{'\n'})
			fileMutex.Unlock()
		}
	} //
}

func ReadCertificate(out chan<- interface{}, filename string, wg *sync.WaitGroup) {
	log.Info("Reading certificates...")
	defer wg.Done()
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		d := json.NewDecoder(file)
		for {
			var f interface{}
			if err := d.Decode(&f); err == io.EOF {
				break
			} else if err != nil {
				// handle error
			}
			out <- f
		}
	} else {
		log.Fatal("Error reading file: ", err)
	}
}

func WriteOutput(in <-chan []byte, outputFileName string, wg *sync.WaitGroup) {
	defer wg.Done()
}

func WriteProcessedFile(in <-chan string, outputFileName string, wg *sync.WaitGroup) {
	defer wg.Done()
}

func main() {
	log.SetLevel(log.InfoLevel)
	runtime.GOMAXPROCS(numProcs)

	//Initialize Channels
	certs := make(chan interface{}, channelSize)
	jsonOut := make(chan []byte, channelSize)
	processOut := make(chan string, channelSize)

	var readerWG sync.WaitGroup
	var procWG sync.WaitGroup
	var writerWG sync.WaitGroup
	var processWg sync.WaitGroup

	readerWG.Add(1)
	writerWG.Add(1)
	processWg.Add(1)

	processFile, err := os.Create(outProcessPath)
	if err != nil {
		log.Fatal("could not create process file")
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatal("could not create output file")
	}

	go ReadCertificate(certs, inPath, &readerWG)
	go WriteOutput(jsonOut, outPath, &writerWG)
	go WriteProcessedFile(processOut, outProcessPath, &processWg)

	for i := 0; i < numCertThreads; i++ {
		procWG.Add(1)
		go ProcessCertificate(certs, jsonOut, outFile, processFile, &procWG)
	}

	go func() {
		readerWG.Wait()
		close(certs)
	}()

	procWG.Wait()
	close(jsonOut)
	writerWG.Wait()
}
