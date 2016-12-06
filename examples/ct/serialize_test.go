package ct

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	ct "github.com/google/certificate-transparency/go"
	"github.com/google/certificate-transparency/go/fixchain"
	"github.com/google/certificate-transparency/go/tls"
	"github.com/google/certificate-transparency/go/x509"
	"github.com/google/trillian/examples/ct/testonly"
)

func TestSignV1SCTForCertificate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cert, err := fixchain.CertificateFromPEM(testonly.LeafSignedByFakeIntermediateCertPem)
	if err != nil {
		t.Fatalf("failed to set up test cert: %v", err)
	}

	km := setupMockKeyManager(mockCtrl, []byte{0x5, 0x62, 0x4f, 0xb4, 0x9e, 0x32, 0x14, 0xb6, 0xc, 0xb8, 0x51, 0x28, 0x23, 0x93, 0x2c, 0x7a, 0x3d, 0x80, 0x93, 0x5f, 0xcd, 0x76, 0xef, 0x91, 0x6a, 0xaf, 0x1b, 0x8c, 0xe8, 0xb5, 0x2, 0xb5})

	leaf, got, err := signV1SCTForCertificate(km, cert, fixedTime)
	if err != nil {
		t.Fatalf("create sct for cert failed: %v", err)
	}

	logID, err := base64.StdEncoding.DecodeString(ctMockLogID)
	if err != nil {
		t.Fatalf("failed to decode test log id: %s", ctMockLogID)
	}

	var idArray [sha256.Size]byte
	copy(idArray[:], logID)

	expected := ct.SignedCertificateTimestamp{
		SCTVersion: 0,
		LogID:      ct.LogID{KeyID: ct.SHA256Hash(idArray)},
		Timestamp:  1504786523000000,
		Extensions: ct.CTExtensions{},
		Signature: ct.DigitallySigned{
			Algorithm: tls.SignatureAndHashAlgorithm{
				Hash:      tls.SHA256,
				Signature: tls.RSA},
			Signature: []byte("signed"),
		},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("Mismatched SCT (cert), got %v, expected %v", got, expected)
	}

	// Additional checks that the MerkleTreeLeaf we built is correct
	if got, want := leaf.Version, ct.V1; got != want {
		t.Fatalf("Got a %v leaf, expected a %v leaf", got, want)
	}
	if got, want := leaf.LeafType, ct.TimestampedEntryLeafType; got != want {
		t.Fatalf("Got leaf type %v, expected %v", got, want)
	}
	if got, want := leaf.TimestampedEntry.EntryType, ct.X509LogEntryType; got != want {
		t.Fatalf("Got entry type %v, expected %v", got, want)
	}
	if got, want := leaf.TimestampedEntry.Timestamp, got.Timestamp; got != want {
		t.Fatalf("Entry / sct timestamp mismatch; got %v, expected %v", got, want)
	}
	if got, want := leaf.TimestampedEntry.X509Entry.Data, cert.Raw; !reflect.DeepEqual(got, want) {
		t.Fatalf("Cert bytes mismatch, got %x, expected %x", got, want)
	}
}

func TestSignV1SCTForPrecertificate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cert, err := fixchain.CertificateFromPEM(testonly.PrecertPEMValid)
	_, ok := err.(x509.NonFatalErrors)

	if err != nil && !ok {
		t.Fatalf("failed to set up test precert: %v", err)
	}

	km := setupMockKeyManager(mockCtrl, []byte{0x77, 0xf3, 0x5c, 0xc6, 0xad, 0x85, 0xfd, 0xe0, 0x38, 0xfd, 0x36, 0x34, 0x5c, 0x1e, 0x45, 0x58, 0x60, 0x95, 0xb1, 0x7c, 0x28, 0xaa, 0xa5, 0xa5, 0x84, 0x96, 0x37, 0x4b, 0xf8, 0xbb, 0xd9, 0x8})

	leaf, got, err := signV1SCTForPrecertificate(km, cert, fixedTime)

	if err != nil {
		t.Fatalf("create sct for precert failed: %v", err)
	}

	logID, err := base64.StdEncoding.DecodeString(ctMockLogID)

	if err != nil {
		t.Fatalf("failed to decode test log id: %s", ctMockLogID)
	}

	var idArray [sha256.Size]byte
	copy(idArray[:], logID)

	expected := ct.SignedCertificateTimestamp{SCTVersion: 0,
		LogID:      ct.LogID{KeyID: ct.SHA256Hash(idArray)},
		Timestamp:  1504786523000000,
		Extensions: ct.CTExtensions{},
		Signature: ct.DigitallySigned{
			Algorithm: tls.SignatureAndHashAlgorithm{
				Hash:      tls.SHA256,
				Signature: tls.RSA},
			Signature: []byte("signed")}}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("Mismatched SCT (precert), got %v, expected %v", got, expected)
	}

	// Additional checks that the MerkleTreeLeaf we built is correct
	keyHash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)

	// Additional checks that the MerkleTreeLeaf we built is correct
	if got, want := leaf.Version, ct.V1; got != want {
		t.Fatalf("Got a %v leaf, expected a %v leaf", got, want)
	}
	if got, want := leaf.LeafType, ct.TimestampedEntryLeafType; got != want {
		t.Fatalf("Got leaf type %v, expected %v", got, want)
	}
	if got, want := leaf.TimestampedEntry.EntryType, ct.PrecertLogEntryType; got != want {
		t.Fatalf("Got entry type %v, expected %v", got, want)
	}
	if got, want := got.Timestamp, leaf.TimestampedEntry.Timestamp; got != want {
		t.Fatalf("Entry / sct timestamp mismatch; got %v, expected %v", got, want)
	}
	if got, want := keyHash[:], leaf.TimestampedEntry.PrecertEntry.IssuerKeyHash[:]; !bytes.Equal(got, want) {
		t.Fatalf("Issuer key hash bytes mismatch, got %v, expected %v", got, want)
	}
	if got, want := leaf.TimestampedEntry.PrecertEntry.TBSCertificate, cert.RawTBSCertificate; !bytes.Equal(got, want) {
		t.Fatalf("TBS cert mismatch, got %v, expected %v", got, want)
	}
}
