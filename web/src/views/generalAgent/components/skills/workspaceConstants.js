export const LANGUAGE_MAP = {
  md: 'markdown',
  py: 'python',
  js: 'javascript',
  ts: 'typescript',
  json: 'json',
  yml: 'yaml',
  yaml: 'yaml',
  txt: 'plaintext',
  sh: 'shell',
  html: 'html',
  css: 'css',
  scss: 'scss',
};

export const MAX_FILE_SIZE_BYTES = 1024 * 1024;

export const getLanguageByPath = path => {
  if (!path) return 'plaintext';
  const ext = path.split('.').pop().toLowerCase();
  return LANGUAGE_MAP[ext] || 'plaintext';
};
