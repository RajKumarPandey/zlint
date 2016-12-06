// lint_cert_policy_requires_org.go
/*If the Certificate asserts the policy identifier of 2.23.140.1.2.2, then it MUST also include organizationName, localityName (to the extent such field is required under Section 7.1.4.2.2), stateOrProvinceName (to the extent such field is required under Section 7.1.4.2.2), and countryName in the Subject field.*/

package lints

import (

	"github.com/teamnsrg/zlint/util"
	"github.com/zmap/zgrab/ztools/x509"
)

type CertPolicyRequiresOrg struct {
	// Internal data here
}

func (l *CertPolicyRequiresOrg) Initialize() error {
	return nil
}

func (l *CertPolicyRequiresOrg) CheckApplies(cert *x509.Certificate) bool {
	return util.SliceContainsOID(cert.PolicyIdentifiers, util.BROrganizationValidatedOID)
}

func (l *CertPolicyRequiresOrg) RunTest(cert *x509.Certificate) (ResultStruct, error) {
	var out ResultStruct
	if util.TypeInName(&cert.Subject, util.OrganizationNameOID) {
		out.Result = Pass
	} else {
		out.Result = Error
	}
	return out, nil
}

func init() {
	RegisterLint(&Lint{
		Name:          "cert_policy_requires_org",
		Description:   "If certificate policy 2.23.140.1.2.2 is included, organizationName must be included in subject.",
		Providence:    "CAB: 7.1.6.1",
		EffectiveDate: util.CABEffectiveDate,
		Test:          &CertPolicyRequiresOrg{}})
}