// 文件图标配置
export const FILE_ICON_CONFIG = {
  md: { icon: 'el-icon-document', color: '#519aba' },
  py: { icon: 'el-icon-document', color: '#3572A5' },
  js: { icon: 'el-icon-document', color: '#f1e05a' },
  ts: { icon: 'el-icon-document', color: '#3178c6' },
  json: { icon: 'el-icon-document', color: '#cbcb41' },
  yml: { icon: 'el-icon-document', color: '#cb171e' },
  yaml: { icon: 'el-icon-document', color: '#cb171e' },
  html: { icon: 'el-icon-document', color: '#e34c26' },
  css: { icon: 'el-icon-document', color: '#563d7c' },
  scss: { icon: 'el-icon-document', color: '#c6538c' },
  sh: { icon: 'el-icon-document', color: '#89e051' },
  txt: { icon: 'el-icon-document', color: '#6d8086' },
};

export const DEFAULT_FILE_ICON = { icon: 'el-icon-document', color: '#6d8086' };

/**
 * 根据文件名获取图标配置
 * @param {string} filename - 文件名
 * @returns {object} 图标配置 { icon, color }
 */
export function getFileIcon(filename) {
  if (!filename) return DEFAULT_FILE_ICON;
  const ext = filename.split('.').pop().toLowerCase();
  return FILE_ICON_CONFIG[ext] || DEFAULT_FILE_ICON;
}
