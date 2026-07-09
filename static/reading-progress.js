(function () {
    var API_BASE = '/api';
    var SAVE_INTERVAL_MS = 8000;
    var MIN_PROGRESS_DELTA = 2;
    var state = {
        user: null,
        lastSavedAt: 0,
        lastSavedProgress: -1,
        saveTimer: null,
        restoring: false
    };

    function ready(fn) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', fn);
        } else {
            fn();
        }
    }

    function request(path, options) {
        options = options || {};
        options.credentials = 'same-origin';
        options.headers = Object.assign({ 'Accept': 'application/json' }, options.headers || {});
        if (options.body && !options.headers['Content-Type']) {
            options.headers['Content-Type'] = 'application/json';
        }
        return fetch(API_BASE + path, options).then(function (resp) {
            return resp.json().catch(function () { return {}; }).then(function (data) {
                if (!resp.ok) {
                    var message = data && data.error && data.error.message ? data.error.message : '请求失败';
                    throw new Error(message);
                }
                return data;
            });
        });
    }

    function getArticlePath() {
        return window.location.pathname || '/';
    }

    function isArticlePage() {
        var path = getArticlePath();
        return path !== '/' &&
            path !== '/index.html' &&
            path !== '/login.html' &&
            path !== '/reading-history.html' &&
            path.indexOf('/static/') !== 0 &&
            path.indexOf('/assets/') !== 0;
    }

    function getArticleTitle() {
        var titleEle = document.getElementById('title');
        if (titleEle && titleEle.textContent.trim()) {
            return titleEle.textContent.trim();
        }
        return document.title || getArticlePath();
    }

    function getProgress() {
        var doc = document.documentElement;
        var body = document.body;
        var scrollY = Math.max(window.scrollY || window.pageYOffset || 0, 0);
        var height = Math.max(doc.scrollHeight, body ? body.scrollHeight : 0);
        var viewport = window.innerHeight || doc.clientHeight || 0;
        var total = height - viewport;
        var progress = total <= 0 ? 100 : Math.round((scrollY / total) * 100);
        progress = Math.max(0, Math.min(100, progress));
        return {
            articlePath: getArticlePath(),
            articleTitle: getArticleTitle(),
            progressPercent: progress,
            scrollY: Math.round(scrollY),
            finished: progress >= 95
        };
    }

    function buildPanel() {
        if (document.getElementById('reading-progress-panel')) {
            return;
        }
        var panel = document.createElement('div');
        panel.id = 'reading-progress-panel';
        panel.className = 'reading-progress-panel';
        panel.innerHTML = '<span id="reading-progress-user">未登录</span>' +
            '<a href="/login.html" id="reading-progress-login">登录</a>' +
            '<a href="/reading-history.html" id="reading-progress-history" style="display:none">阅读记录</a>' +
            '<button id="reading-progress-logout" type="button" style="display:none">退出</button>';
        document.body.appendChild(panel);

        var logout = document.getElementById('reading-progress-logout');
        logout.addEventListener('click', function () {
            request('/auth/logout', { method: 'POST' }).then(function () {
                window.location.reload();
            }).catch(function (err) {
                alert(err.message);
            });
        });
    }

    function renderAuth() {
        var userText = document.getElementById('reading-progress-user');
        var login = document.getElementById('reading-progress-login');
        var history = document.getElementById('reading-progress-history');
        var logout = document.getElementById('reading-progress-logout');
        if (!userText || !login || !history || !logout) {
            return;
        }
        if (state.user) {
            userText.textContent = state.user.displayName || state.user.username;
            login.style.display = 'none';
            history.style.display = '';
            logout.style.display = '';
        } else {
            userText.textContent = '未登录';
            login.style.display = '';
            history.style.display = 'none';
            logout.style.display = 'none';
        }
    }

    function loadCurrentUser() {
        return request('/auth/me').then(function (data) {
            state.user = data.authenticated ? data.user : null;
            renderAuth();
            return state.user;
        }).catch(function () {
            state.user = null;
            renderAuth();
            return null;
        });
    }

    function maybeRestore() {
        if (!state.user || !isArticlePage()) {
            return;
        }
        var articlePath = encodeURIComponent(getArticlePath());
        request('/reading-progress?articlePath=' + articlePath).then(function (data) {
            if (!data.found || !data.item || !data.item.scrollY || data.item.progressPercent <= 0) {
                return;
            }
            showRestoreTip(data.item);
        }).catch(function () {});
    }

    function showRestoreTip(item) {
        if (document.getElementById('reading-restore-tip')) {
            return;
        }
        var tip = document.createElement('div');
        tip.id = 'reading-restore-tip';
        tip.className = 'reading-restore-tip';
        tip.innerHTML = '<span>上次读到 ' + item.progressPercent + '%，是否继续？</span>' +
            '<button type="button" id="reading-restore-yes">继续阅读</button>' +
            '<button type="button" id="reading-restore-no">忽略</button>';
        document.body.appendChild(tip);

        document.getElementById('reading-restore-yes').addEventListener('click', function () {
            state.restoring = true;
            window.scrollTo(0, item.scrollY || 0);
            setTimeout(function () { state.restoring = false; }, 600);
            tip.parentNode.removeChild(tip);
        });
        document.getElementById('reading-restore-no').addEventListener('click', function () {
            tip.parentNode.removeChild(tip);
        });
    }

    function scheduleSave(force) {
        if (!state.user || !isArticlePage() || state.restoring) {
            return;
        }
        if (state.saveTimer) {
            clearTimeout(state.saveTimer);
        }
        state.saveTimer = setTimeout(function () {
            saveProgress(force);
        }, force ? 0 : 500);
    }

    function saveProgress(force) {
        if (!state.user || !isArticlePage()) {
            return;
        }
        var now = Date.now();
        var payload = getProgress();
        if (!force) {
            if (now - state.lastSavedAt < SAVE_INTERVAL_MS) {
                return;
            }
            if (Math.abs(payload.progressPercent - state.lastSavedProgress) < MIN_PROGRESS_DELTA) {
                return;
            }
        }
        state.lastSavedAt = now;
        state.lastSavedProgress = payload.progressPercent;
        request('/reading-progress', {
            method: 'PUT',
            keepalive: force,
            body: JSON.stringify(payload)
        }).catch(function () {});
    }

    ready(function () {
        buildPanel();
        loadCurrentUser().then(function () {
            maybeRestore();
            if (state.user && isArticlePage()) {
                scheduleSave(true);
                window.addEventListener('scroll', function () { scheduleSave(false); }, { passive: true });
                window.addEventListener('pagehide', function () { saveProgress(true); });
                window.addEventListener('beforeunload', function () { saveProgress(true); });
            }
        });
    });
})();
