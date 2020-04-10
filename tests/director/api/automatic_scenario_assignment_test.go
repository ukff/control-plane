package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
)

func TestAutomaticScenarioAssignmentQueries(t *testing.T) {
	// GIVEN
	ctx := context.Background()
	tenantID := testTenants.GetIDByName(t, "ASA1")

	testScenarioA := "ASA1"
	testScenarioB := "ASA2"
	testScenarioC := "ASA3"
	testSelectorA := graphql.LabelSelectorInput{
		Key:   "keyA",
		Value: "valueA",
	}
	testSelectorB := graphql.LabelSelectorInput{
		Key:   "keyB",
		Value: "valueB",
	}
	testSelectorAGQL, err := tc.graphqlizer.LabelSelectorInputToGQL(testSelectorA)
	require.NoError(t, err)

	// setup available scenarios
	createScenariosLabelDefinitionWithinTenant(t, ctx, tenantID, []string{"DEFAULT", testScenarioA, testScenarioB, testScenarioC})

	// create automatic scenario assignments
	inputAssignment1 := graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: testScenarioA,
		Selector:     &testSelectorA,
	}
	inputAssignment2 := graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: testScenarioB,
		Selector:     &testSelectorA,
	}
	inputAssignment3 := graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: testScenarioC,
		Selector:     &testSelectorB,
	}
	createAutomaticScenarioAssignmentInTenant(t, ctx, inputAssignment1, tenantID)
	defer deleteAutomaticScenarioAssignmentForScenarioWithinTenant(t, ctx, tenantID, testScenarioA)
	createAutomaticScenarioAssignmentInTenant(t, ctx, inputAssignment2, tenantID)
	defer deleteAutomaticScenarioAssignmentForScenarioWithinTenant(t, ctx, tenantID, testScenarioB)
	createAutomaticScenarioAssignmentInTenant(t, ctx, inputAssignment3, tenantID)
	defer deleteAutomaticScenarioAssignmentForScenarioWithinTenant(t, ctx, tenantID, testScenarioC)

	// prepare queries
	getAssignmentForScenarioRequest := fixAutomaticScenarioAssignmentForScenarioRequest(testScenarioA)
	listAssignmentsRequest := fixAutomaticScenarioAssignmentsRequest()
	listAssignmentsForSelectorRequest := fixAutomaticScenarioAssignmentsForSelectorRequest(testSelectorAGQL)

	actualAssignmentsPage := graphql.AutomaticScenarioAssignmentPage{}
	actualAssignmentForScenario := graphql.AutomaticScenarioAssignment{}
	actualAssignmentsForSelector := []*graphql.AutomaticScenarioAssignment{}

	// WHEN
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, listAssignmentsRequest, &actualAssignmentsPage)
	require.NoError(t, err)
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, getAssignmentForScenarioRequest, &actualAssignmentForScenario)
	require.NoError(t, err)
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, listAssignmentsForSelectorRequest, &actualAssignmentsForSelector)
	require.NoError(t, err)

	// THEN
	saveExample(t, listAssignmentsRequest.Query(), "query automatic scenario assignments")
	saveExample(t, getAssignmentForScenarioRequest.Query(), "query automatic scenario assignment for scenario")
	saveExample(t, listAssignmentsForSelectorRequest.Query(), "query automatic scenario assignments for selector")

	assertAutomaticScenarioAssignments(t,
		[]graphql.AutomaticScenarioAssignmentSetInput{inputAssignment1, inputAssignment2, inputAssignment3},
		actualAssignmentsPage.Data)
	assertAutomaticScenarioAssignment(t, inputAssignment1, actualAssignmentForScenario)
	assertAutomaticScenarioAssignments(t,
		[]graphql.AutomaticScenarioAssignmentSetInput{inputAssignment1, inputAssignment2},
		actualAssignmentsForSelector)
}

func Test_AutomaticScenarioAssigmentForRuntime(t *testing.T) {
	//GIVEN
	ctx := context.TODO()
	tenantID := testTenants.GetIDByName(t, "TestCreateAutomaticScenarioAssignment")
	prodScenario := "PRODUCTION"
	manualScenario := "MANUAL"
	defaultScenario := "DEFAULT"
	createScenariosLabelDefinitionWithinTenant(t, ctx, tenantID, []string{prodScenario, manualScenario, defaultScenario})

	rtms := make([]*graphql.RuntimeExt, 3)
	for i := 0; i < 3; i++ {
		rmtInput := fixRuntimeInput(fmt.Sprintf("runtime%d", i))

		rtm := registerRuntimeFromInputWithinTenant(t, ctx, &rmtInput, tenantID)
		rtms[i] = rtm
		defer unregisterRuntimeWithinTenant(t, rtm.ID, tenantID)
	}

	selectorKey := "KEY"
	selectorValue := "VALUE"

	setRuntimeLabelWithinTenant(t, ctx, tenantID, rtms[0].ID, selectorKey, selectorValue)
	setRuntimeLabelWithinTenant(t, ctx, tenantID, rtms[1].ID, selectorKey, selectorValue)

	t.Run("Check automatic scenario assigment", func(t *testing.T) {
		//GIVEN
		expectedScenarios := map[string][]interface{}{
			rtms[0].ID: {defaultScenario, prodScenario},
			rtms[1].ID: {defaultScenario, prodScenario},
			rtms[2].ID: {defaultScenario},
		}

		//WHEN
		asaInput := fixAutomaticScenarioAssigmentInput(prodScenario, selectorKey, selectorValue)
		createAutomaticScenarioAssignmentInTenant(t, ctx, asaInput, tenantID)
		defer deleteAutomaticScenarioAssigmentForSelector(t, ctx, tenantID, *asaInput.Selector)

		//THEN
		runtimes := listRuntimes(t, ctx, tenantID)
		require.Len(t, runtimes.Data, 3)
		assertRuntimeScenarios(t, runtimes, expectedScenarios)
	})
}

func assertRuntimeScenarios(t *testing.T, runtimes graphql.RuntimePageExt, expectedScenarios map[string][]interface{}) {
	for _, rtm := range runtimes.Data {
		expectedScenarios, found := expectedScenarios[rtm.ID]
		require.True(t, found)
		assertScenarios(t, rtm.Labels, expectedScenarios)
	}
}

func assertScenarios(t *testing.T, actual graphql.Labels, expected []interface{}) {
	val, ok := actual["scenarios"]
	require.True(t, ok)
	scenarios, ok := val.([]interface{})
	require.True(t, ok)
	assert.ElementsMatch(t, scenarios, expected)
}

func Test_DeleteAutomaticScenarioAssignmentForScenario(t *testing.T) {
	//GIVEN
	ctx := context.Background()

	defaultValue := "DEFAULT"
	scenario1 := "test-scenario"
	scenario2 := "test-scenario-2"
	selector := &graphql.LabelSelectorInput{
		Value: "test-value",
		Key:   "test-key",
	}

	scenarios := []string{defaultValue, scenario1, scenario2}
	tenantID := testTenants.GetIDByName(t, "TestDeleteAssignmentsForScenario")
	createScenariosLabelDefinitionWithinTenant(t, ctx, tenantID, scenarios)

	assignment1 := graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: scenario1,
		Selector:     selector,
	}
	assignment2 := graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: scenario2,
		Selector:     selector,
	}

	var output graphql.AutomaticScenarioAssignment

	assignment1Gql, err := tc.graphqlizer.AutomaticScenarioAssignmentSetInputToGQL(assignment1)
	require.NoError(t, err)

	req := fixCreateAutomaticScenarioAssignmentRequest(assignment1Gql)
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, req, nil)
	require.NoError(t, err)
	saveExample(t, req.Query(), "create automatic scenario assignment")

	createAutomaticScenarioAssignmentInTenant(t, ctx, assignment2, tenantID)
	defer deleteAutomaticScenarioAssignmentForScenarioWithinTenant(t, ctx, tenantID, scenario2)

	//WHEN
	req = fixDeleteAutomaticScenarioAssignmentForScenarioRequest(scenario1)
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, req, &output)
	require.NoError(t, err)

	//THEN
	assertAutomaticScenarioAssignment(t, assignment1, output)

	allAssignments := listAutomaticScenarioAssignmentsWithinTenant(t, ctx, tenantID)
	require.Len(t, allAssignments.Data, 1)
	require.Equal(t, 1, allAssignments.TotalCount)
	assertAutomaticScenarioAssignment(t, assignment2, *allAssignments.Data[0])

	saveExample(t, req.Query(), "delete automatic scenario assignment for scenario")
}

func Test_DeleteAutomaticScenarioAssignmentForSelector(t *testing.T) {
	//GIVEN
	ctx := context.Background()
	defaultValue := "DEFAULT"
	scenario1 := "test-scenario"
	scenario2 := "test-scenario-2"
	scenario3 := "test-scenario-3"

	scenarios := []string{defaultValue, scenario1, scenario2, scenario3}

	tenantID := testTenants.GetIDByName(t, "TestDeleteAssignmentsForSelector")
	createScenariosLabelDefinitionWithinTenant(t, ctx, tenantID, scenarios)

	selector := graphql.LabelSelectorInput{Key: "test-key", Value: "test-value"}
	selector2 := graphql.LabelSelectorInput{
		Key:   "test-key-2",
		Value: "test-value-2",
	}

	assignments := []graphql.AutomaticScenarioAssignmentSetInput{
		{ScenarioName: scenario1, Selector: &selector},
		{ScenarioName: scenario2, Selector: &selector},
	}
	anotherAssignment := graphql.AutomaticScenarioAssignmentSetInput{ScenarioName: scenario3, Selector: &selector2}

	var output []*graphql.AutomaticScenarioAssignment

	createAutomaticScenarioAssignmentInTenant(t, ctx, assignments[0], tenantID)
	createAutomaticScenarioAssignmentInTenant(t, ctx, assignments[1], tenantID)
	createAutomaticScenarioAssignmentInTenant(t, ctx, anotherAssignment, tenantID)
	defer deleteAutomaticScenarioAssignmentForScenarioWithinTenant(t, ctx, tenantID, scenario3)

	selectorGql, err := tc.graphqlizer.LabelSelectorInputToGQL(selector)
	require.NoError(t, err)

	//WHEN
	req := fixDeleteAutomaticScenarioAssignmentsForSelectorRequest(selectorGql)
	err = tc.RunOperationWithCustomTenant(ctx, tenantID, req, &output)
	require.NoError(t, err)

	//THEN
	assertAutomaticScenarioAssignments(t, assignments, output)

	actualAssignments := listAutomaticScenarioAssignmentsWithinTenant(t, ctx, tenantID)
	assert.Len(t, actualAssignments.Data, 1)
	require.Equal(t, 1, actualAssignments.TotalCount)
	assertAutomaticScenarioAssignment(t, anotherAssignment, *actualAssignments.Data[0])

	saveExample(t, req.Query(), "delete automatic scenario assignments for selector")

}

func fixAutomaticScenarioAssigmentInput(automaticScenario, selecterKey, selectorValue string) graphql.AutomaticScenarioAssignmentSetInput {
	return graphql.AutomaticScenarioAssignmentSetInput{
		ScenarioName: automaticScenario,
		Selector: &graphql.LabelSelectorInput{
			Key:   selecterKey,
			Value: selectorValue,
		},
	}

}
