/*
 * ZLint Copyright 2018 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package lints

import (
	"github.com/zmap/zcrypto/x509"
	"github.com/zmap/zlint/util"
)

type IssuerDNLeadingSpace struct{}

func (l *IssuerDNLeadingSpace) Initialize() error {
	return nil
}

func (l *IssuerDNLeadingSpace) CheckApplies(c *x509.Certificate) bool {
	return true
}

func (l *IssuerDNLeadingSpace) Execute(c *x509.Certificate) *LintResult {
	leading, _, err := util.CheckRDNSequenceWhiteSpace(c.RawIssuer)
	if err != nil {
		return &LintResult{Status: Fatal}
	}
	if leading {
		return &LintResult{Status: Warn}
	}
	return &LintResult{Status: Pass}
}

func init() {
	RegisterLint(&Lint{
		Name:          "w_issuer_dn_leading_whitespace",
		Description:   "AttributeValue in issuer RelativeDistinguishedName sequence SHOULD NOT have leading whitespace",
		Citation:      "AWSLabs certlint",
		Source:        AWSLabs,
		EffectiveDate: util.ZeroDate,
		Lint:          &IssuerDNLeadingSpace{},
	})
}
