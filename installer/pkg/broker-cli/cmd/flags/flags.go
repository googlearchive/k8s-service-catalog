// Copyright © 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Package flags is a wrapper around the pflags library which adds support
// for required flags. You should use the *VarP functions just like with the
// pflags package except that it's a function instead of a method. Then, you
// can call CheckRequiredFlags to see whether any desired flags are missing.

package flags

import (
	"fmt"
	"log"

	"github.com/spf13/pflag"
)

// Names keeps the long and short flag names.
type Names struct {
	long  string
	short string
}

var (
	// nameMap is a map of flag pointers to their Names.
	nameMap = make(map[interface{}]Names)
)

// StringArrayFlag is a wrapper to *FlagSet.StringArrayVarP but does some additional
// stuff so that you can use CheckRequiredFlags with the given pointer p.
func StringArrayFlag(flagset *pflag.FlagSet, p *[]string, long, short, usage string) {
	if p == nil {
		log.Fatal("nil pointer given to StringArrayVarP? This should never happen")
	}
	nameMap[p] = Names{short: short, long: long}
	flagset.StringArrayVarP(p, long, short, nil, usage)
}

// StringFlag is a wrapper to *FlagSet.StringVarP which also does some
// book keeping so that the flag can be used with GetShortName and GetLongName.
func StringFlag(flagset *pflag.FlagSet, p *string, long, short, usage string) {
	if p == nil {
		log.Fatal("nil pointer given to RequiredStringVarP? This should never happen")
	}
	nameMap[p] = Names{short: short, long: long}
	flagset.StringVarP(p, long, short, "", usage)
}

// BoolFlag is a wrapper to *FlagSet.BoolVarP which also does some book keeping so that the flag can
// be used with GetShortName and GetLongName.
func BoolFlag(flagset *pflag.FlagSet, p *bool, long, short, usage string) {
	if p == nil {
		log.Fatal("nil pointer given to RequiredBoolVarP? This should never happen")
	}
	nameMap[p] = Names{short: short, long: long}
	flagset.BoolVarP(p, long, short, false /* bool flags default to FALSE */, usage)
}

// CheckRequiredFlags will check that all required flags are there.
// You must use this package's *VarP functions instead of the *FlagSet methods
// for all the flags that you are passed into this function.
// Note that requiredFlags are not actually flags but rather pointers to the variables which are
// set by the user flags.
func CheckRequiredFlags(requiredFlags ...interface{}) []Names {
	missingFlagNames := make([]Names, 0)
	for _, requiredFlag := range requiredFlags {
		if requiredFlag == nil {
			log.Fatal("Null flag given? This should never happen..")
		} else {
			switch typedFlag := requiredFlag.(type) {
			case *string:
				if *typedFlag == "" {
					missingFlagNames = addFlagName(missingFlagNames, typedFlag)
				}
			case *[]string:
				if *typedFlag == nil {
					missingFlagNames = addFlagName(missingFlagNames, typedFlag)
				}
			default:
				log.Fatalf("Unknown type in CheckRequiredFlags: %v", typedFlag)
			}
		}
	}
	if len(missingFlagNames) > 0 {
		return missingFlagNames
	} else {
		return nil
	}
}

// Prints error message to user about missing flags.
func PrintMissingFlags(missingFlags []Names) {
	fmt.Printf("Missing required flags: %+v\nPlease use -h/--help to find more details about the flags", missingFlags)
}

func addFlagName(missingFlagNames []Names, flag interface{}) []Names {
	names, ok := nameMap[flag]
	if !ok {
		// If you see this then you used this function without calling the appropriate *Flag function from this package.
		log.Fatal("Required flag not in nameMap? This should never happen")
	}
	return append(missingFlagNames, names)
}

// String returns string representation of Name for pretty printing.
// Looks like (-, --longFlag).
func (names Names) String() string {
	return fmt.Sprintf("(-%s, --%s)", names.short, names.long)
}
