package url_test

import (
	"gitlab.com/yakshaving.art/git-pull-mirror/url"
	"reflect"
	"testing"
)

func Test_ParseGitHubURL(t *testing.T) {
	tt := []struct {
		name             string
		url              string
		expectedPath     string
		expectedOwner    string
		expectedName     string
		expectedUsername string
		expectedPassword string
		expectedError    error
	}{
		{
			"Invalid URL",
			"",
			"",
			"",
			"",
			"",
			"",
			url.ErrInvalidURL,
		},
		{
			"GitHub Git URL",
			"git@github.com:gomeeseeks/meeseeks-box.git",
			"github.com/gomeeseeks/meeseeks-box",
			"gomeeseeks",
			"meeseeks-box",
			"git",
			"",
			nil,
		},
		{
			"GitHub Git URL without ending",
			"git@github.com:gomeeseeks/meeseeks-box",
			"github.com/gomeeseeks/meeseeks-box",
			"gomeeseeks",
			"meeseeks-box",
			"git",
			"",
			nil,
		},
		{
			"GitHub HTTP URL",
			"https://github.com/gomeeseeks/meeseeks-box.git",
			"github.com/gomeeseeks/meeseeks-box",
			"gomeeseeks",
			"meeseeks-box",
			"",
			"",
			nil,
		},
		{
			"GitHub HTTP URL",
			"http://github.com/gomeeseeks/meeseeks-box.git",
			"github.com/gomeeseeks/meeseeks-box",
			"gomeeseeks",
			"meeseeks-box",
			"",
			"",
			nil,
		},
		{
			"GitHub HTTP URL without ending",
			"http://github.com/gomeeseeks/meeseeks-box",
			"github.com/gomeeseeks/meeseeks-box",
			"gomeeseeks",
			"meeseeks-box",
			"",
			"",
			nil,
		},
		{
			"GitLab URL with nested groups with ending",
			"http://gitlab.com/group/subgroup/project.git",
			"gitlab.com/group/subgroup/project",
			"group",
			"subgroup/project",
			"",
			"",
			nil,
		},
		{
			"GitLab URL with nested groups without ending",
			"http://gitlab.com/group/subgroup/project",
			"gitlab.com/group/subgroup/project",
			"group",
			"subgroup/project",
			"",
			"",
			nil,
		},
		{
			"GitLab URL with username and password",
			"http://username:password@gitlab.com/group/project",
			"gitlab.com/group/project",
			"group",
			"project",
			"username",
			"password",
			nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			url, err := url.Parse(tc.url)
			assertEquals(t, tc.expectedError, err)
			if err == nil {
				assertEquals(t, tc.expectedPath, url.ToPath())
				assertEquals(t, tc.expectedOwner, url.Owner)
				assertEquals(t, tc.expectedName, url.Name)
				assertEquals(t, tc.expectedUsername, url.Username)
				assertEquals(t, tc.expectedPassword, url.Password)
			}
		})
	}
}

func assertEquals(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Value %s is not as expected %s", actual, expected)
	}
}

func assertErr(t *testing.T, expected, actual error) {
	if expected != nil && actual != nil {
		assertEquals(t, expected.Error(), actual.Error())
	} else if expected != nil || actual != nil {
		t.Fatalf("Error %s is not as expected %s", actual, expected)
	}
}
