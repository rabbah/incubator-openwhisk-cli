/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"errors"
	"fmt"

	"../wski18n"
	"github.com/apache/incubator-openwhisk-client-go/whisk"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const FEED_LIFECYCLE_EVENT = "lifecycleEvent"
const FEED_TRIGGER_NAME = "triggerName"
const FEED_AUTH_KEY = "authKey"
const FEED_CREATE = "CREATE"
const FEED_READ = "READ"
const FEED_UPDATE = "UPDATE"
const FEED_DELETE = "DELETE"

// triggerCmd represents the trigger command
var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: wski18n.T("work with triggers"),
}

var triggerFireCmd = &cobra.Command{
	Use:           "fire TRIGGER_NAME [PAYLOAD]",
	Short:         wski18n.T("fire trigger event"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var parameters interface{}
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 1, 2, "Trigger fire",
			wski18n.T("A trigger name is required. A payload is optional.")); whiskErr != nil {
			return whiskErr
		}

		if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
			return NewQualifiedNameError(args[0], err)
		}

		Client.Namespace = qualifiedName.GetNamespace()

		// Add payload to parameters
		if len(args) == 2 {
			Flags.common.param = append(Flags.common.param, getFormattedJSON("payload", args[1]))
			Flags.common.param = append(Flags.common.param, Flags.common.param...)
		}

		if len(Flags.common.param) > 0 {
			parameters, err = getJSONFromStrings(Flags.common.param, false)
			if err != nil {
				whisk.Debug(whisk.DbgError, "getJSONFromStrings(%#v, false) failed: %s\n", Flags.common.param, err)
				errStr := wski18n.T("Invalid parameter argument '{{.param}}': {{.err}}",
					map[string]interface{}{"param": fmt.Sprintf("%#v", Flags.common.param), "err": err})
				werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL,
					whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
				return werr
			}
		}

		trigResp, _, err := Client.Triggers.Fire(qualifiedName.GetEntityName(), parameters)
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Fire(%s, %#v) failed: %s\n", qualifiedName.GetEntityName(), parameters, err)
			errStr := wski18n.T("Unable to fire trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL,
				whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		fmt.Fprintf(color.Output,
			wski18n.T("{{.ok}} triggered /{{.namespace}}/{{.name}} with id {{.id}}\n",
				map[string]interface{}{
					"ok":        color.GreenString("ok:"),
					"namespace": boldString(qualifiedName.GetNamespace()),
					"name":      boldString(qualifiedName.GetEntityName()),
					"id":        boldString(trigResp.ActivationId)}))
		return nil
	},
}

var triggerCreateCmd = &cobra.Command{
	Use:           "create TRIGGER_NAME",
	Short:         wski18n.T("create new trigger"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var annotations interface{}
		var feedArgPassed bool = (Flags.common.feed != "")
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 1, 1, "Trigger create",
			wski18n.T("A trigger name is required.")); whiskErr != nil {
			return whiskErr
		}

		if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
			return NewQualifiedNameError(args[0], err)
		}

		Client.Namespace = qualifiedName.GetNamespace()

		var fullTriggerName string
		var fullFeedName string
		var feedQualifiedName = new(QualifiedName)
		if feedArgPassed {
			whisk.Debug(whisk.DbgInfo, "Trigger has a feed\n")

			if feedQualifiedName, err = NewQualifiedName(Flags.common.feed); err != nil {
				return NewQualifiedNameError(Flags.common.feed, err)
			}

			fullFeedName = fmt.Sprintf("/%s/%s", feedQualifiedName.GetNamespace(), feedQualifiedName.GetEntityName())
			fullTriggerName = fmt.Sprintf("/%s/%s", qualifiedName.GetNamespace(), qualifiedName.GetEntityName())
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_LIFECYCLE_EVENT, FEED_CREATE))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_TRIGGER_NAME, fullTriggerName))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_AUTH_KEY, Client.Config.AuthToken))
		}

		// Convert the trigger's list of default parameters from a string into []KeyValue
		// The 1 or more --param arguments have all been combined into a single []string
		// e.g.   --p arg1,arg2 --p arg3,arg4   ->  [arg1, arg2, arg3, arg4]
		whisk.Debug(whisk.DbgInfo, "Parsing parameters: %#v\n", Flags.common.param)
		parameters, err := getJSONFromStrings(Flags.common.param, !feedArgPassed)

		if err != nil {
			whisk.Debug(whisk.DbgError, "getJSONFromStrings(%#v, true) failed: %s\n", Flags.common.param, err)
			errStr := wski18n.T("Invalid parameter argument '{{.param}}': {{.err}}",
				map[string]interface{}{"param": fmt.Sprintf("%#v", Flags.common.param), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
			return werr
		}

		// Add feed to annotations
		if feedArgPassed {
			Flags.common.annotation = append(Flags.common.annotation, getFormattedJSON("feed", Flags.common.feed))
		}

		whisk.Debug(whisk.DbgInfo, "Parsing annotations: %#v\n", Flags.common.annotation)
		annotations, err = getJSONFromStrings(Flags.common.annotation, true)

		if err != nil {
			whisk.Debug(whisk.DbgError, "getJSONFromStrings(%#v, true) failed: %s\n", Flags.common.annotation, err)
			errStr := wski18n.T("Invalid annotation argument '{{.annotation}}': {{.err}}",
				map[string]interface{}{"annotation": fmt.Sprintf("%#v", Flags.common.annotation), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
			return werr
		}

		trigger := &whisk.Trigger{
			Name:        qualifiedName.GetEntityName(),
			Annotations: annotations.(whisk.KeyValueArr),
		}

		if !feedArgPassed {
			trigger.Parameters = parameters.(whisk.KeyValueArr)
		}

		_, _, err = Client.Triggers.Insert(trigger, false)
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Insert(%+v,false) failed: %s\n", trigger, err)
			errStr := wski18n.T("Unable to create trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": trigger.Name, "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		// Invoke the specified feed action to configure the trigger feed
		if feedArgPassed {
			err := configureFeed(trigger.Name, fullFeedName)
			if err != nil {
				whisk.Debug(whisk.DbgError, "configureFeed(%s, %s) failed: %s\n", trigger.Name, Flags.common.feed,
					err)
				errStr := wski18n.T("Unable to create trigger '{{.name}}': {{.err}}",
					map[string]interface{}{"name": trigger.Name, "err": err})
				werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)

				// Delete trigger that was created for this feed
				delerr := deleteTrigger(args[0])
				if delerr != nil {
					whisk.Debug(whisk.DbgWarn, "Ignoring deleteTrigger(%s) failure: %s\n", args[0], delerr)
				}
				return werr
			}
		}

		fmt.Fprintf(color.Output,
			wski18n.T("{{.ok}} created trigger {{.name}}\n",
				map[string]interface{}{"ok": color.GreenString("ok:"), "name": boldString(trigger.Name)}))
		return nil
	},
}

var triggerUpdateCmd = &cobra.Command{
	Use:           "update TRIGGER_NAME",
	Short:         wski18n.T("update an existing trigger, or create a trigger if it does not exist"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var fullFeedName string
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 1, 1, "Trigger update",
			wski18n.T("A trigger name is required.")); whiskErr != nil {
			return whiskErr
		}

		if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
			return NewQualifiedNameError(args[0], err)
		}

		Client.Namespace = qualifiedName.GetNamespace()

		// Convert the trigger's list of default parameters from a string into []KeyValue
		// The 1 or more --param arguments have all been combined into a single []string
		// e.g.   --p arg1,arg2 --p arg3,arg4   ->  [arg1, arg2, arg3, arg4]

		whisk.Debug(whisk.DbgInfo, "Parsing parameters: %#v\n", Flags.common.param)
		parameters, err := getJSONFromStrings(Flags.common.param, true)

		if err != nil {
			whisk.Debug(whisk.DbgError, "getJSONFromStrings(%#v, true) failed: %s\n", Flags.common.param, err)
			errStr := wski18n.T("Invalid parameter argument '{{.param}}': {{.err}}",
				map[string]interface{}{"param": fmt.Sprintf("%#v", Flags.common.param), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
			return werr
		}

		whisk.Debug(whisk.DbgInfo, "Parsing annotations: %#v\n", Flags.common.annotation)
		annotations, err := getJSONFromStrings(Flags.common.annotation, true)

		if err != nil {
			whisk.Debug(whisk.DbgError, "getJSONFromStrings(%#v, true) failed: %s\n", Flags.common.annotation, err)
			errStr := wski18n.T("Invalid annotation argument '{{.annotation}}': {{.err}}",
				map[string]interface{}{"annotation": fmt.Sprintf("%#v", Flags.common.annotation), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
			return werr
		}

		retTrigger, _, err := Client.Triggers.Get(qualifiedName.GetEntityName())
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Get(%s) failed: %s\n", qualifiedName.GetEntityName(), err)
			errStr := wski18n.T("Unable to get trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		// Get full feed name from trigger get request as it is needed to get the feed
		if retTrigger != nil && retTrigger.Annotations != nil {
			fullFeedName = getValueString(retTrigger.Annotations, "feed")
		}

		if len(fullFeedName) > 0 {
			fullTriggerName := fmt.Sprintf("/%s/%s", qualifiedName.GetNamespace(), qualifiedName.GetEntityName())
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_LIFECYCLE_EVENT, FEED_UPDATE))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_TRIGGER_NAME, fullTriggerName))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_AUTH_KEY, Client.Config.AuthToken))

			// Invoke the specified feed action to configure the trigger feed
			err = configureFeed(qualifiedName.GetEntityName(), fullFeedName)
			if err != nil {
				whisk.Debug(whisk.DbgError, "configureFeed(%s, %s) failed: %s\n", qualifiedName.GetEntityName(), Flags.common.feed,
					err)
				errStr := wski18n.T("Unable to create trigger '{{.name}}': {{.err}}",
					map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
				werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
				return werr
			}
		} else {
			trigger := &whisk.Trigger{
				Name:        qualifiedName.GetEntityName(),
				Parameters:  parameters.(whisk.KeyValueArr),
				Annotations: annotations.(whisk.KeyValueArr),
			}

			_, _, err = Client.Triggers.Insert(trigger, true)
			if err != nil {
				whisk.Debug(whisk.DbgError, "Client.Triggers.Insert(%+v,true) failed: %s\n", trigger, err)
				errStr := wski18n.T("Unable to update trigger '{{.name}}': {{.err}}",
					map[string]interface{}{"name": trigger.Name, "err": err})
				werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
				return werr
			}
		}

		fmt.Fprintf(color.Output,
			wski18n.T("{{.ok}} updated trigger {{.name}}\n",
				map[string]interface{}{"ok": color.GreenString("ok:"), "name": boldString(qualifiedName.GetEntityName())}))
		return nil
	},
}

var triggerGetCmd = &cobra.Command{
	Use:           "get TRIGGER_NAME [FIELD_FILTER]",
	Short:         wski18n.T("get trigger"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var field string
		var fullFeedName string
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 1, 2, "Trigger get", wski18n.T("A trigger name is required.")); whiskErr != nil {
			return whiskErr
		}

		if len(args) > 1 {
			field = args[1]

			if !fieldExists(&whisk.Trigger{}, field) {
				errMsg := wski18n.T("Invalid field filter '{{.arg}}'.", map[string]interface{}{"arg": field})
				whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXIT_CODE_ERR_GENERAL,
					whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
				return whiskErr
			}
		}

		if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
			return NewQualifiedNameError(args[0], err)
		}

		Client.Namespace = qualifiedName.GetNamespace()

		retTrigger, _, err := Client.Triggers.Get(qualifiedName.GetEntityName())
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Get(%s) failed: %s\n", qualifiedName.GetEntityName(), err)
			errStr := wski18n.T("Unable to get trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		// Get full feed name from trigger get request as it is needed to get the feed
		if retTrigger != nil && retTrigger.Annotations != nil {
			fullFeedName = getValueString(retTrigger.Annotations, "feed")
		}

		if len(fullFeedName) > 0 {
			fullTriggerName := fmt.Sprintf("/%s/%s", qualifiedName.GetNamespace(), qualifiedName.GetEntityName())
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_LIFECYCLE_EVENT, FEED_READ))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_TRIGGER_NAME, fullTriggerName))
			Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_AUTH_KEY, Client.Config.AuthToken))

			err = configureFeed(qualifiedName.GetEntityName(), fullFeedName)
			if err != nil {
				whisk.Debug(whisk.DbgError, "configureFeed(%s, %s) failed: %s\n", qualifiedName.GetEntityName(), fullFeedName, err)
			}
		} else {
			if Flags.trigger.summary {
				printSummary(retTrigger)
			} else {
				if len(field) > 0 {
					fmt.Fprintf(color.Output, wski18n.T("{{.ok}} got trigger {{.name}}, displaying field {{.field}}\n",
						map[string]interface{}{"ok": color.GreenString("ok:"), "name": boldString(qualifiedName.GetEntityName()),
							"field": boldString(field)}))
					printField(retTrigger, field)
				} else {
					fmt.Fprintf(color.Output, wski18n.T("{{.ok}} got trigger {{.name}}\n",
						map[string]interface{}{"ok": color.GreenString("ok:"), "name": boldString(qualifiedName.GetEntityName())}))
					printJSON(retTrigger)
				}
			}
		}

		return nil
	},
}

var triggerDeleteCmd = &cobra.Command{
	Use:           "delete TRIGGER_NAME",
	Short:         wski18n.T("delete trigger"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var retTrigger *whisk.Trigger
		var fullFeedName string
		var origParams []string
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 1, 1, "Trigger delete",
			wski18n.T("A trigger name is required.")); whiskErr != nil {
			return whiskErr
		}

		if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
			return NewQualifiedNameError(args[0], err)
		}

		Client.Namespace = qualifiedName.GetNamespace()

		retTrigger, _, err = Client.Triggers.Get(qualifiedName.GetEntityName())
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Get(%s) failed: %s\n", qualifiedName.GetEntityName(), err)
			errStr := wski18n.T("Unable to get trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		// Get full feed name from trigger delete request as it is needed to delete the feed
		if retTrigger != nil && retTrigger.Annotations != nil {
			fullFeedName = getValueString(retTrigger.Annotations, "feed")

			if len(fullFeedName) > 0 {
				origParams = Flags.common.param
				fullTriggerName := fmt.Sprintf("/%s/%s", qualifiedName.GetNamespace(), qualifiedName.GetEntityName())
				Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_LIFECYCLE_EVENT, FEED_DELETE))
				Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_TRIGGER_NAME, fullTriggerName))
				Flags.common.param = append(Flags.common.param, getFormattedJSON(FEED_AUTH_KEY, Client.Config.AuthToken))

				err = configureFeed(qualifiedName.GetEntityName(), fullFeedName)
				if err != nil {
					whisk.Debug(whisk.DbgError, "configureFeed(%s, %s) failed: %s\n", qualifiedName.GetEntityName(), fullFeedName, err)
				}

				Flags.common.param = origParams
				Client.Namespace = qualifiedName.GetNamespace()
			}

		}

		retTrigger, _, err = Client.Triggers.Delete(qualifiedName.GetEntityName())
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.Delete(%s) failed: %s\n", qualifiedName.GetEntityName(), err)
			errStr := wski18n.T("Unable to delete trigger '{{.name}}': {{.err}}",
				map[string]interface{}{"name": qualifiedName.GetEntityName(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}

		fmt.Fprintf(color.Output,
			wski18n.T("{{.ok}} deleted trigger {{.name}}\n",
				map[string]interface{}{"ok": color.GreenString("ok:"), "name": boldString(qualifiedName.GetEntityName())}))

		return nil
	},
}

var triggerListCmd = &cobra.Command{
	Use:           "list [NAMESPACE]",
	Short:         wski18n.T("list all triggers"),
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE:       SetupClientConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var qualifiedName = new(QualifiedName)

		if whiskErr := CheckArgs(args, 0, 1, "Trigger list",
			wski18n.T("An optional namespace is the only valid argument.")); whiskErr != nil {
			return whiskErr
		}

		if len(args) == 1 {
			if qualifiedName, err = NewQualifiedName(args[0]); err != nil {
				return NewQualifiedNameError(args[0], err)
			}

			if len(qualifiedName.GetEntityName()) > 0 {
				return entityNameError(qualifiedName.GetEntityName())
			}

			Client.Namespace = qualifiedName.GetNamespace()
		}

		options := &whisk.TriggerListOptions{
			Skip:  Flags.common.skip,
			Limit: Flags.common.limit,
		}
		triggers, _, err := Client.Triggers.List(options)
		if err != nil {
			whisk.Debug(whisk.DbgError, "Client.Triggers.List(%#v) for namespace '%s' failed: %s\n", options,
				Client.Namespace, err)
			errStr := wski18n.T("Unable to obtain the list of triggers for namespace '{{.name}}': {{.err}}",
				map[string]interface{}{"name": getClientNamespace(), "err": err})
			werr := whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
			return werr
		}
		sortByName := Flags.common.nameSort
		printList(triggers, sortByName)
		return nil
	},
}

func configureFeed(triggerName string, FullFeedName string) error {
	feedArgs := []string{FullFeedName}
	Flags.common.blocking = true
	err := actionInvokeCmd.RunE(nil, feedArgs)
	if err != nil {
		whisk.Debug(whisk.DbgError, "Invoke of action '%s' failed: %s\n", FullFeedName, err)
		errStr := wski18n.T("Unable to invoke trigger '{{.trigname}}' feed action '{{.feedname}}'; feed is not configured: {{.err}}",
			map[string]interface{}{"trigname": triggerName, "feedname": FullFeedName, "err": err})
		err = whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
	} else {
		whisk.Debug(whisk.DbgInfo, "Successfully configured trigger feed via feed action '%s'\n", FullFeedName)
	}

	return err
}

func deleteTrigger(triggerName string) error {
	args := []string{triggerName}
	err := triggerDeleteCmd.RunE(nil, args)
	if err != nil {
		whisk.Debug(whisk.DbgError, "Trigger '%s' delete failed: %s\n", triggerName, err)
		errStr := wski18n.T("Unable to delete trigger '{{.name}}': {{.err}}",
			map[string]interface{}{"name": triggerName, "err": err})
		err = whisk.MakeWskErrorFromWskError(errors.New(errStr), err, whisk.EXIT_CODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
	}

	return err
}

func init() {
	triggerCreateCmd.Flags().StringSliceVarP(&Flags.common.annotation, "annotation", "a", []string{}, wski18n.T("annotation values in `KEY VALUE` format"))
	triggerCreateCmd.Flags().StringVarP(&Flags.common.annotFile, "annotation-file", "A", "", wski18n.T("`FILE` containing annotation values in JSON format"))
	triggerCreateCmd.Flags().StringSliceVarP(&Flags.common.param, "param", "p", []string{}, wski18n.T("parameter values in `KEY VALUE` format"))
	triggerCreateCmd.Flags().StringVarP(&Flags.common.paramFile, "param-file", "P", "", wski18n.T("`FILE` containing parameter values in JSON format"))
	triggerCreateCmd.Flags().StringVarP(&Flags.common.feed, "feed", "f", "", wski18n.T("trigger feed `ACTION_NAME`"))

	triggerUpdateCmd.Flags().StringSliceVarP(&Flags.common.annotation, "annotation", "a", []string{}, wski18n.T("annotation values in `KEY VALUE` format"))
	triggerUpdateCmd.Flags().StringVarP(&Flags.common.annotFile, "annotation-file", "A", "", wski18n.T("`FILE` containing annotation values in JSON format"))
	triggerUpdateCmd.Flags().StringSliceVarP(&Flags.common.param, "param", "p", []string{}, wski18n.T("parameter values in `KEY VALUE` format"))
	triggerUpdateCmd.Flags().StringVarP(&Flags.common.paramFile, "param-file", "P", "", wski18n.T("`FILE` containing parameter values in JSON format"))

	triggerGetCmd.Flags().BoolVarP(&Flags.trigger.summary, "summary", "s", false, wski18n.T("summarize trigger details; parameters with prefix \"*\" are bound"))

	triggerFireCmd.Flags().StringSliceVarP(&Flags.common.param, "param", "p", []string{}, wski18n.T("parameter values in `KEY VALUE` format"))
	triggerFireCmd.Flags().StringVarP(&Flags.common.paramFile, "param-file", "P", "", wski18n.T("`FILE` containing parameter values in JSON format"))

	triggerListCmd.Flags().IntVarP(&Flags.common.skip, "skip", "s", 0, wski18n.T("exclude the first `SKIP` number of triggers from the result"))
	triggerListCmd.Flags().IntVarP(&Flags.common.limit, "limit", "l", 30, wski18n.T("only return `LIMIT` number of triggers from the collection"))
	triggerListCmd.Flags().BoolVarP(&Flags.common.nameSort, "name-sort", "n", false, wski18n.T("sorts a list alphabetically by entity name; only applicable within the limit/skip returned entity block"))

	triggerCmd.AddCommand(
		triggerFireCmd,
		triggerCreateCmd,
		triggerUpdateCmd,
		triggerGetCmd,
		triggerDeleteCmd,
		triggerListCmd,
	)

}
