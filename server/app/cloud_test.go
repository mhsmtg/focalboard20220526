package app

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mmModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"

	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/focalboard/server/services/store"
)

func TestIsCloud(t *testing.T) {
	t.Run("if it's not running on plugin mode", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.Store.EXPECT().GetLicense().Return(nil)
		require.False(t, th.App.IsCloud())
	})

	t.Run("if it's running on plugin mode but the license is incomplete", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		fakeLicense := &mmModel.License{}

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		require.False(t, th.App.IsCloud())

		fakeLicense = &mmModel.License{Features: &mmModel.Features{}}

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		require.False(t, th.App.IsCloud())
	})

	t.Run("if it's running on plugin mode, with a non-cloud license", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		fakeLicense := &mmModel.License{
			Features: &mmModel.Features{Cloud: mmModel.NewBool(false)},
		}

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		require.False(t, th.App.IsCloud())
	})

	t.Run("if it's running on plugin mode with a cloud license", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		fakeLicense := &mmModel.License{
			Features: &mmModel.Features{Cloud: mmModel.NewBool(true)},
		}

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		require.True(t, th.App.IsCloud())
	})
}

func TestIsCloudLimited(t *testing.T) {
	t.Run("if no limit has been set, it should be false", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		require.Zero(t, th.App.CardLimit)
		require.False(t, th.App.IsCloudLimited())
	})

	t.Run("if the limit is set, it should be true", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		fakeLicense := &mmModel.License{
			Features: &mmModel.Features{Cloud: mmModel.NewBool(true)},
		}
		th.Store.EXPECT().GetLicense().Return(fakeLicense)

		th.App.CardLimit = 5
		require.True(t, th.App.IsCloudLimited())
	})
}

func TestSetCloudLimits(t *testing.T) {
	t.Run("if the limits are empty, it should do nothing", func(t *testing.T) {
		t.Run("limits empty", func(t *testing.T) {
			th, tearDown := SetupTestHelper(t)
			defer tearDown()

			require.Zero(t, th.App.CardLimit)

			require.NoError(t, th.App.SetCloudLimits(nil))
			require.Zero(t, th.App.CardLimit)
		})

		t.Run("limits not empty but board limits empty", func(t *testing.T) {
			th, tearDown := SetupTestHelper(t)
			defer tearDown()

			require.Zero(t, th.App.CardLimit)

			limits := &mmModel.ProductLimits{}

			require.NoError(t, th.App.SetCloudLimits(limits))
			require.Zero(t, th.App.CardLimit)
		})
	})

	t.Run("if the limits are not empty, it should update them and calculate the new timestamp", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		require.Zero(t, th.App.CardLimit)

		newCardLimitTimestamp := int64(27)
		th.Store.EXPECT().UpdateCardLimitTimestamp(5).Return(newCardLimitTimestamp, nil)

		limits := &mmModel.ProductLimits{
			Boards: &mmModel.BoardsLimits{Cards: mmModel.NewInt(5)},
		}

		require.NoError(t, th.App.SetCloudLimits(limits))
		require.Equal(t, 5, th.App.CardLimit)
	})

	t.Run("if the limits are already set and we unset them, the timestamp will be unset too", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.App.CardLimit = 20

		th.Store.EXPECT().UpdateCardLimitTimestamp(0)

		require.NoError(t, th.App.SetCloudLimits(nil))

		require.Zero(t, th.App.CardLimit)
	})

	t.Run("if the limits are already set and we try to set the same ones again", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.App.CardLimit = 20

		// the call to update card limit timestamp should not happen
		// as the limits didn't change
		th.Store.EXPECT().UpdateCardLimitTimestamp(gomock.Any()).Times(0)

		limits := &mmModel.ProductLimits{
			Boards: &mmModel.BoardsLimits{Cards: mmModel.NewInt(20)},
		}

		require.NoError(t, th.App.SetCloudLimits(limits))
		require.Equal(t, 20, th.App.CardLimit)
	})
}

func TestUpdateCardLimitTimestamp(t *testing.T) {
	fakeLicense := &mmModel.License{
		Features: &mmModel.Features{Cloud: mmModel.NewBool(true)},
	}

	t.Run("if the server is a cloud instance but not limited, it should do nothing", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		require.Zero(t, th.App.CardLimit)

		// the license check will not be done as the limit not being
		// set is enough for the method to return
		th.Store.EXPECT().GetLicense().Times(0)
		// no call to UpdateCardLimitTimestamp should happen as the
		// method should shortcircuit if not cloud limited
		th.Store.EXPECT().UpdateCardLimitTimestamp(gomock.Any()).Times(0)

		require.NoError(t, th.App.UpdateCardLimitTimestamp())
	})

	t.Run("if the server is a cloud instance and the timestamp is set, it should run the update", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.App.CardLimit = 5

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		// no call to UpdateCardLimitTimestamp should happen as the
		// method should shortcircuit if not cloud limited
		th.Store.EXPECT().UpdateCardLimitTimestamp(5)

		require.NoError(t, th.App.UpdateCardLimitTimestamp())
	})
}

func TestNotifyPortalAdminsUpgradeRequest(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	t.Run("should send message", func(t *testing.T) {
		pluginAPI := &plugintest.API{}

		sysAdmin1 := &mmModel.User{
			Id:       "michael-scott",
			Username: "Michael Scott",
		}

		sysAdmin2 := &mmModel.User{
			Id:       "dwight-schrute",
			Username: "Dwight Schrute",
		}

		getUsersOptionsPage0 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    0,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage0).Return([]*mmModel.User{sysAdmin1, sysAdmin2}, nil).Once()

		getUsersOptionsPage1 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    1,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage1).Return([]*mmModel.User{}, nil).Once()

		th.App.pluginAPI = pluginAPI

		team := &mmModel.Team{
			DisplayName: "Dunder Mifflin",
		}

		th.Store.EXPECT().GetWorkspaceTeam("team-id-1").Return(team, nil)
		th.Store.EXPECT().SendMessage(gomock.Any(), "custom_cloud_upgrade_nudge", gomock.Any()).Return(nil).Times(1)

		err := th.App.NotifyPortalAdminsUpgradeRequest("team-id-1")
		assert.NoError(t, err)
	})

	t.Run("no sys admins found", func(t *testing.T) {
		pluginAPI := &plugintest.API{}

		getUsersOptionsPage0 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    0,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage0).Return([]*mmModel.User{}, nil).Once()

		th.App.pluginAPI = pluginAPI

		team := &mmModel.Team{
			DisplayName: "Dunder Mifflin",
		}

		th.Store.EXPECT().GetWorkspaceTeam("team-id-1").Return(team, nil)

		err := th.App.NotifyPortalAdminsUpgradeRequest("team-id-1")
		assert.NoError(t, err)
	})

	t.Run("iterate multiple pages", func(t *testing.T) {
		pluginAPI := &plugintest.API{}

		sysAdmin1 := &mmModel.User{
			Id:       "michael-scott",
			Username: "Michael Scott",
		}

		sysAdmin2 := &mmModel.User{
			Id:       "dwight-schrute",
			Username: "Dwight Schrute",
		}

		getUsersOptionsPage0 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    0,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage0).Return([]*mmModel.User{sysAdmin1}, nil).Once()

		getUsersOptionsPage1 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    1,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage1).Return([]*mmModel.User{sysAdmin2}, nil).Once()

		getUsersOptionsPage2 := &mmModel.UserGetOptions{
			Active:  true,
			Role:    mmModel.SystemAdminRoleId,
			PerPage: 50,
			Page:    2,
		}
		pluginAPI.On("GetUsers", getUsersOptionsPage2).Return([]*mmModel.User{}, nil).Once()

		th.App.pluginAPI = pluginAPI

		team := &mmModel.Team{
			DisplayName: "Dunder Mifflin",
		}

		th.Store.EXPECT().GetWorkspaceTeam("team-id-1").Return(team, nil)
		th.Store.EXPECT().SendMessage(gomock.Any(), "custom_cloud_upgrade_nudge", gomock.Any()).Return(nil).Times(2)

		err := th.App.NotifyPortalAdminsUpgradeRequest("team-id-1")
		assert.NoError(t, err)
	})
}

func TestGetTemplateMapForBlocks(t *testing.T) {
	container := store.Container{
		WorkspaceID: "0",
	}

	t.Run("should not access the database if all boards are present already in the blocks", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "board1",
				Type:     model.TypeBoard,
				ParentID: "board1",
				RootID:   "board1",
				Fields:   map[string]interface{}{"isTemplate": true},
			},
			{
				ID:       "card1",
				Type:     model.TypeCard,
				ParentID: "board1",
				RootID:   "board1",
			},
			{
				ID:       "board2",
				Type:     model.TypeBoard,
				ParentID: "board2",
				RootID:   "board2",
				Fields:   map[string]interface{}{"isTemplate": false},
			},
			{
				ID:       "card2",
				Type:     model.TypeCard,
				ParentID: "board2",
				RootID:   "board2",
			},
			{
				ID:       "text2",
				Type:     model.TypeText,
				ParentID: "card2",
				RootID:   "board2",
			},
		}

		templateMap, err := th.App.getTemplateMapForBlocks(container, blocks)
		require.NoError(t, err)
		require.Len(t, templateMap, 2)
		require.Contains(t, templateMap, "board1")
		require.True(t, templateMap["board1"])
		require.Contains(t, templateMap, "board2")
		require.False(t, templateMap["board2"])
	})

	t.Run("should fetch boards from the database if not present", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "board1",
				Type:     model.TypeBoard,
				ParentID: "board1",
				RootID:   "board1",
				Fields:   map[string]interface{}{"isTemplate": true},
			},
			{
				ID:       "card1",
				Type:     model.TypeCard,
				ParentID: "board1",
				RootID:   "board1",
			},
			{
				ID:       "card2",
				Type:     model.TypeCard,
				ParentID: "board2",
				RootID:   "board2",
			},
			{
				ID:       "text3",
				Type:     model.TypeText,
				ParentID: "card3",
				RootID:   "board3",
			},
		}

		// doesn't have Fields, so it should be treated as not
		// template
		board2 := model.Block{
			ID:       "board2",
			Type:     model.TypeBoard,
			ParentID: "board2",
			RootID:   "board2",
		}

		board3 := model.Block{
			ID:       "board3",
			Type:     model.TypeBoard,
			ParentID: "board3",
			RootID:   "board3",
			Fields:   map[string]interface{}{"isTemplate": true},
		}

		th.Store.EXPECT().
			GetBlocksByIDs(container, gomock.InAnyOrder([]string{"board2", "board3"})).
			Return([]model.Block{board2, board3}, nil)

		templateMap, err := th.App.getTemplateMapForBlocks(container, blocks)
		require.NoError(t, err)
		require.Len(t, templateMap, 3)
		require.Contains(t, templateMap, "board1")
		require.True(t, templateMap["board1"])
		require.Contains(t, templateMap, "board2")
		require.False(t, templateMap["board2"])
		require.Contains(t, templateMap, "board3")
		require.True(t, templateMap["board3"])
	})
}

func TestApplyCloudLimits(t *testing.T) {
	container := store.Container{
		WorkspaceID: "0",
	}

	fakeLicense := &mmModel.License{
		Features: &mmModel.Features{Cloud: mmModel.NewBool(true)},
	}

	blocks := []model.Block{
		{
			ID:       "board1",
			Type:     model.TypeBoard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 100,
		},
		{
			ID:       "card1",
			Type:     model.TypeCard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 100,
		},
		{
			ID:       "text1",
			Type:     model.TypeText,
			ParentID: "card1",
			RootID:   "board1",
			UpdateAt: 100,
		},
		{
			ID:       "card2",
			Type:     model.TypeCard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 200,
		},
		{
			ID:       "template",
			Type:     model.TypeBoard,
			ParentID: "template",
			RootID:   "template",
			UpdateAt: 1,
			Fields:   map[string]interface{}{"isTemplate": true},
		},
		{
			ID:       "card-from-template",
			Type:     model.TypeCard,
			ParentID: "template",
			RootID:   "template",
			UpdateAt: 1,
		},
	}

	t.Run("if the server is not limited, it should return the blocks untouched", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		require.Zero(t, th.App.CardLimit)

		newBlocks, err := th.App.ApplyCloudLimits(container, blocks)
		require.NoError(t, err)
		require.ElementsMatch(t, blocks, newBlocks)
	})

	t.Run("if the server is limited, it should limit the blocks that are beyond the card limit timestamp", func(t *testing.T) {
		findBlock := func(blocks []model.Block, id string) model.Block {
			for _, block := range blocks {
				if block.ID == id {
					return block
				}
			}
			require.FailNow(t, "block %s not found", id)
			return model.Block{} // this should never be reached
		}

		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		th.App.CardLimit = 5

		th.Store.EXPECT().GetLicense().Return(fakeLicense)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(int64(150), nil)

		newBlocks, err := th.App.ApplyCloudLimits(container, blocks)
		require.NoError(t, err)

		// boards are never limited
		require.False(t, findBlock(newBlocks, "board1").Limited)
		// should be limited as it's beyond the threshold
		require.True(t, findBlock(newBlocks, "card1").Limited)
		// only cards are limited
		require.False(t, findBlock(newBlocks, "text1").Limited)
		// should not be limited as it's not beyond the threshold
		require.False(t, findBlock(newBlocks, "card2").Limited)
		// cards belonging to templates are never limited
		require.False(t, findBlock(newBlocks, "template").Limited)
		require.False(t, findBlock(newBlocks, "card-from-template").Limited)
	})
}

func TestContainsLimitedBlocks(t *testing.T) {
	container := store.Container{
		WorkspaceID: "0",
	}

	// for all the following tests, the timestamp will be set to 150,
	// which means that blocks with an UpdateAt set to 100 will be
	// outside the active window and possibly limited, and blocks with
	// UpdateAt set to 200 will not

	t.Run("should return false if the card limit timestamp is zero", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "card1",
				Type:     model.TypeCard,
				ParentID: "board1",
				RootID:   "board1",
				UpdateAt: 100,
			},
		}

		th.Store.EXPECT().GetCardLimitTimestamp().Return(int64(0), nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.False(t, containsLimitedBlocks)
	})

	t.Run("should return true if the block set contains a card that is limited", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "card1",
				Type:     model.TypeCard,
				ParentID: "board1",
				RootID:   "board1",
				UpdateAt: 100,
			},
		}

		board1 := model.Block{
			ID:       "board1",
			Type:     model.TypeBoard,
			ParentID: "board1",
			RootID:   "board1",
		}

		th.App.CardLimit = 500
		cardLimitTimestamp := int64(150)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(cardLimitTimestamp, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"board1"}).Return([]model.Block{board1}, nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.True(t, containsLimitedBlocks)
	})

	t.Run("should return false if that same block belongs to a template", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "card1",
				Type:     model.TypeCard,
				ParentID: "board1",
				RootID:   "board1",
				UpdateAt: 100,
			},
		}

		board1 := model.Block{
			ID:       "board1",
			Type:     model.TypeBoard,
			ParentID: "board1",
			RootID:   "board1",
			Fields:   map[string]interface{}{"isTemplate": true},
		}

		th.App.CardLimit = 500
		cardLimitTimestamp := int64(150)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(cardLimitTimestamp, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"board1"}).Return([]model.Block{board1}, nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.False(t, containsLimitedBlocks)
	})

	t.Run("should return true if the block contains a content block that belongs to a card that should be limited", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "text1",
				Type:     model.TypeText,
				ParentID: "card1",
				RootID:   "board1",
				UpdateAt: 200,
			},
		}

		card1 := model.Block{
			ID:       "card1",
			Type:     model.TypeCard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 100,
		}

		board1 := model.Block{
			ID:       "board1",
			Type:     model.TypeBoard,
			ParentID: "board1",
			RootID:   "board1",
		}

		th.App.CardLimit = 500
		cardLimitTimestamp := int64(150)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(cardLimitTimestamp, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"card1"}).Return([]model.Block{card1}, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"board1"}).Return([]model.Block{board1}, nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.True(t, containsLimitedBlocks)
	})

	t.Run("should return false if that same block belongs to a card that is inside the active window", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "text1",
				Type:     model.TypeText,
				ParentID: "card1",
				RootID:   "board1",
				UpdateAt: 200,
			},
		}

		card1 := model.Block{
			ID:       "card1",
			Type:     model.TypeCard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 200,
		}

		board1 := model.Block{
			ID:       "board1",
			Type:     model.TypeBoard,
			ParentID: "board1",
			RootID:   "board1",
		}

		th.App.CardLimit = 500
		cardLimitTimestamp := int64(150)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(cardLimitTimestamp, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"card1"}).Return([]model.Block{card1}, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, []string{"board1"}).Return([]model.Block{board1}, nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.False(t, containsLimitedBlocks)
	})

	t.Run("should reach to the database to fetch the necessary information only in an efficient way", func(t *testing.T) {
		th, tearDown := SetupTestHelper(t)
		defer tearDown()

		blocks := []model.Block{
			{
				ID:       "board1",
				Type:     model.TypeBoard,
				ParentID: "board1",
				RootID:   "board1",
			},
			// a content block that references a card that needs
			// fetching but a present board
			{
				ID:       "text1",
				Type:     model.TypeText,
				ParentID: "card1",
				RootID:   "board1",
				UpdateAt: 100,
			},
			// a board that needs fetching referenced by a card and a content block
			{
				ID:       "card2",
				Type:     model.TypeCard,
				ParentID: "board2",
				RootID:   "board2",
				// per timestamp should be limited but the board is a
				// template
				UpdateAt: 100,
			},
			{
				ID:       "text2",
				Type:     model.TypeText,
				ParentID: "card2",
				RootID:   "board2",
				UpdateAt: 200,
			},
			// a content block that references a card and a board,
			// both absent
			{
				ID:       "image3",
				Type:     model.TypeImage,
				ParentID: "card3",
				RootID:   "board3",
				UpdateAt: 100,
			},
		}

		card1 := model.Block{
			ID:       "card1",
			Type:     model.TypeCard,
			ParentID: "board1",
			RootID:   "board1",
			UpdateAt: 200,
		}

		card3 := model.Block{
			ID:       "card3",
			Type:     model.TypeCard,
			ParentID: "board3",
			RootID:   "board3",
			UpdateAt: 200,
		}

		board2 := model.Block{
			ID:       "board2",
			Type:     model.TypeBoard,
			ParentID: "board2",
			RootID:   "board2",
			Fields:   map[string]interface{}{"isTemplate": true},
		}

		board3 := model.Block{
			ID:       "board3",
			Type:     model.TypeBoard,
			ParentID: "board3",
			RootID:   "board3",
		}

		th.App.CardLimit = 500
		cardLimitTimestamp := int64(150)
		th.Store.EXPECT().GetCardLimitTimestamp().Return(cardLimitTimestamp, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, gomock.InAnyOrder([]string{"card1", "card3"})).Return([]model.Block{card1, card3}, nil)
		th.Store.EXPECT().GetBlocksByIDs(container, gomock.InAnyOrder([]string{"board2", "board3"})).Return([]model.Block{board2, board3}, nil)

		containsLimitedBlocks, err := th.App.ContainsLimitedBlocks(container, blocks)
		require.NoError(t, err)
		require.False(t, containsLimitedBlocks)
	})
}
