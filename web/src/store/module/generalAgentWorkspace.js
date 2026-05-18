/**
 * Workspace Vuex Store Module
 * 管理工作空间状态
 */

const state = {
  // 当前活跃的 workspace 信息
  activeWorkspace: null,
  // { runId, threadId, fileCount, totalSize, timestamp }

  // workspace 文件树缓存
  workspaceTrees: {},
  // { [`${threadId}-${runId}`]: { files, fileCount, totalSize, loaded, loading } }

  // 面板状态
  panelVisible: false,
  panelLoading: false,

  // 当前选中的 runId
  currentRunId: null,
};

const getters = {
  activeWorkspace: state => state.activeWorkspace,
  panelVisible: state => state.panelVisible,
  panelLoading: state => state.panelLoading,
  currentRunId: state => state.currentRunId,

  // 获取当前工作空间文件树
  currentWorkspaceTree: state => {
    if (!state.activeWorkspace) return null;
    const key = `${state.activeWorkspace.threadId}-${state.activeWorkspace.runId}`;
    return state.workspaceTrees[key] || null;
  },

  // 是否有工作空间数据
  hasWorkspace: state => {
    return state.activeWorkspace && state.activeWorkspace.fileCount > 0;
  },
};

const mutations = {
  // 设置活跃的工作空间
  SET_ACTIVE_WORKSPACE(state, payload) {
    state.activeWorkspace = payload;
    state.currentRunId = payload ? payload.runId || '' : null;
  },

  // 更新工作空间文件树
  SET_WORKSPACE_TREE(state, { threadId, runId, data }) {
    const key = `${threadId}-${runId}`;
    state.workspaceTrees = {
      ...state.workspaceTrees,
      [key]: {
        files: data.files || [],
        fileCount: data.fileCount || 0,
        totalSize: data.totalSize || 0,
        isDisplay: data.isDisplay || false,
        loaded: true,
        loading: false,
      },
    };
  },

  // 设置加载状态
  SET_WORKSPACE_LOADING(state, { threadId, runId, loading }) {
    const key = `${threadId}-${runId}`;
    const existing = state.workspaceTrees[key] || {};
    state.workspaceTrees = {
      ...state.workspaceTrees,
      [key]: {
        ...existing,
        loading,
      },
    };
  },

  // 切换面板显示
  TOGGLE_PANEL(state, visible) {
    state.panelVisible = visible !== undefined ? visible : !state.panelVisible;
  },

  // 设置面板加载状态
  SET_PANEL_LOADING(state, loading) {
    state.panelLoading = loading;
  },

  // 清除工作空间
  CLEAR_WORKSPACE(state) {
    state.activeWorkspace = null;
    state.currentRunId = null;
  },

  // 重置状态
  RESET_STATE(state) {
    state.activeWorkspace = null;
    state.workspaceTrees = {};
    state.panelVisible = false;
    state.panelLoading = false;
    state.currentRunId = null;
  },
};

const actions = {
  // 处理 workspace activity 事件
  handleWorkspaceActivity({ commit, state }, content) {
    if (!content) return;

    const { runId, threadId, fileCount, totalSize, timestamp } = content;

    // 更新活跃工作空间
    commit('SET_ACTIVE_WORKSPACE', {
      runId,
      threadId,
      fileCount: fileCount || 0,
      totalSize: totalSize || 0,
      timestamp: timestamp || Date.now(),
    });

    // 如果面板已打开，自动刷新文件树
    if (state.panelVisible) {
      return { shouldRefresh: true, threadId, runId };
    }

    return { shouldRefresh: false, threadId, runId };
  },

  // 设置活跃工作空间（用于点击卡片时）
  setActiveWorkspace({ commit }, payload) {
    commit('SET_ACTIVE_WORKSPACE', payload);
  },

  // 显示面板
  showPanel({ commit }) {
    commit('TOGGLE_PANEL', true);
  },

  // 隐藏面板
  hidePanel({ commit }) {
    commit('TOGGLE_PANEL', false);
  },

  // 切换面板
  togglePanel({ commit, state }) {
    commit('TOGGLE_PANEL', !state.panelVisible);
  },

  // 设置工作空间文件树数据
  setWorkspaceTree({ commit }, { threadId, runId, data }) {
    commit('SET_WORKSPACE_TREE', { threadId, runId, data });
  },

  // 设置加载状态
  setWorkspaceLoading({ commit }, { threadId, runId, loading }) {
    commit('SET_WORKSPACE_LOADING', { threadId, runId, loading });
  },

  // 清除当前工作空间
  clearWorkspace({ commit }) {
    commit('CLEAR_WORKSPACE');
  },

  // 重置
  reset({ commit }) {
    commit('RESET_STATE');
  },
};

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions,
};
