package cdx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/ebay/sbom-scorecard/pkg/scorecard"
)

type CycloneDXReport struct {
	valid    bool
	docError error

	creationToolName    int
	creationToolVersion int

	totalPackages  int
	hasLicense     int
	hasPackVersion int
	hasPackDigest  int
	hasPurl        int
	hasCPE         int
}

func (r *CycloneDXReport) Report() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d total packages\n", r.totalPackages))

	sb.WriteString(fmt.Sprintf("%d%% have versions.\n", scorecard.PrettyPercent(r.hasPackVersion, r.totalPackages)))
	sb.WriteString(fmt.Sprintf("%d%% have licenses.\n", scorecard.PrettyPercent(r.hasLicense, r.totalPackages)))
	sb.WriteString(fmt.Sprintf("%d%% have package digest.\n", scorecard.PrettyPercent(r.hasPackDigest, r.totalPackages)))
	sb.WriteString(fmt.Sprintf("%d%% have purls.\n", scorecard.PrettyPercent(r.hasPurl, r.totalPackages)))
	sb.WriteString(fmt.Sprintf("%d%% have CPEs.\n", scorecard.PrettyPercent(r.hasCPE, r.totalPackages)))

	sb.WriteString(fmt.Sprintf("Has creation info? %v\n", r.hasCreationInfo()))
	sb.WriteString(fmt.Sprintf("Spec valid? %v\n", r.valid))
	return sb.String()
}

func (r *CycloneDXReport) hasCreationInfo() bool {
	return r.creationToolName > 0 &&
		r.creationToolVersion > 0 &&
		r.creationToolName == r.creationToolVersion
}

func (r *CycloneDXReport) IsSpecCompliant() scorecard.ReportValue {
	if r.docError != nil {
		return scorecard.ReportValue{
			Ratio:     0,
			Reasoning: r.docError.Error(),
		}
	}
	return scorecard.ReportValue{Ratio: 1}
}

func (r *CycloneDXReport) PackageIdentification() scorecard.ReportValue {
	purlPercent := scorecard.PrettyPercent(r.hasPurl, r.totalPackages)
	cpePercent := scorecard.PrettyPercent(r.hasCPE, r.totalPackages)
	return scorecard.ReportValue{
		// What percentage has both Purl & CPEs?
		Ratio:     float32(r.hasPurl+r.hasCPE) / float32(r.totalPackages*2),
		Reasoning: fmt.Sprintf("%d%% have purls and %d%% have CPEs", purlPercent, cpePercent),
	}
}

func (r *CycloneDXReport) PackageVersions() scorecard.ReportValue {
	return scorecard.ReportValue{
		Ratio: float32(r.hasPackVersion) / float32(r.totalPackages),
	}
}

func (r *CycloneDXReport) PackageDigests() scorecard.ReportValue {
	return scorecard.ReportValue{
		Ratio: float32(r.hasPackDigest) / float32(r.totalPackages),
	}
}

func (r *CycloneDXReport) PackageLicenses() scorecard.ReportValue {
	return scorecard.ReportValue{
		Ratio: float32(r.hasLicense) / float32(r.totalPackages),
	}
}

func (r *CycloneDXReport) CreationInfo() scorecard.ReportValue {
	// @@@
	return scorecard.ReportValue{Ratio: 1}
}

func GetCycloneDXReport(filename string) scorecard.SbomReport {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error while opening %v for reading: %v", filename, err)
		return nil
	}

	r := CycloneDXReport{}
	formats := []cdx.BOMFileFormat{cdx.BOMFileFormatJSON, cdx.BOMFileFormatXML}

	bom := new(cdx.BOM)
	for _, format := range formats {
		decoder := cdx.NewBOMDecoder(bytes.NewReader(contents), format)
		if err = decoder.Decode(bom); err != nil {
			r.valid = false
			r.docError = err
		} else {
			r.valid = true
			r.docError = nil
			break
		}
	}

	if !r.valid {
		return &r
	}

	if bom.Metadata.Tools != nil {
		for _, t := range *bom.Metadata.Tools {
			if t.Name != "" {
				r.creationToolName += 1
			}
			if t.Version != "" {
				r.creationToolVersion += 1
			}
		}
	}

	if bom.Components != nil {
		for _, p := range *bom.Components {
			r.totalPackages += 1
			if p.Licenses != nil && len(*p.Licenses) > 0 {
				r.hasLicense += 1
			}
			if p.Hashes != nil && len(*p.Hashes) > 0 {
				r.hasPackDigest += 1
			}
			if p.Version != "" {
				r.hasPackVersion += 1
			}
			if p.PackageURL != "" {
				r.hasPurl += 1
			}
			if p.CPE != "" {
				r.hasCPE += 1
			}
		}
	}

	return &r
}
