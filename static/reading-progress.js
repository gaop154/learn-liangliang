(function () {
    var API_BASE = '/api';
    var SAVE_INTERVAL_MS = 8000;
    var MIN_PROGRESS_DELTA = 2;
    var state = {
        user: null,
        lastSavedAt: 0,
        lastSavedProgress: -1,
        highestProgress: -1,
        highestScrollY: 0,
        saveTimer: null,
        restoring: false,
        otherCategoryArticles: {}
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

    function canonicalArticlePath(rawPath) {
        var articlePath = (rawPath || '/').trim();
        try {
            articlePath = decodeURIComponent(articlePath);
        } catch (e) {
            return '';
        }
        articlePath = articlePath.replace(/\\/g, '/');
        if (articlePath.indexOf('\0') !== -1 || articlePath.length > 1024) {
            return '';
        }
        if (articlePath.charAt(0) !== '/') {
            articlePath = '/' + articlePath;
        }
        var segments = articlePath.split('/');
        var normalizedSegments = [];
        for (var i = 0; i < segments.length; i++) {
            if (!segments[i] || segments[i] === '.') {
                continue;
            }
            if (segments[i] === '..') {
                return '';
            }
            normalizedSegments.push(segments[i]);
        }
        articlePath = '/' + normalizedSegments.join('/');
        if (articlePath === '/content') {
            articlePath = '/';
        } else if (articlePath.indexOf('/content/') === 0) {
            articlePath = articlePath.substring('/content'.length);
        }
        return articlePath;
    }

    function pathFromLink(link, basePath) {
        try {
            var target = new URL(link.getAttribute('href'), basePath || window.location.origin);
            if (target.origin !== window.location.origin) {
                return '';
            }
            return canonicalArticlePath(target.pathname);
        } catch (e) {
            return '';
        }
    }

    function getArticlePath() {
        return canonicalArticlePath(window.location.pathname || '/');
    }

    function isCourseArticlePath(articlePath, coursePath) {
        if (!/^\/专栏\/[^/]+\/[^/]+\.md\.html$/.test(articlePath)) {
            return false;
        }
        return !coursePath || articlePath.indexOf(coursePath + '/') === 0;
    }

    function isOtherArticlePath(articlePath) {
        return /^\/其他\/(?:恋爱必修课|文章|极客时间)\/[^/]+\.md\.html$/.test(articlePath);
    }

    function isArticlePath(articlePath) {
        return isCourseArticlePath(articlePath) || isOtherArticlePath(articlePath);
    }

    function getCoursePath(articlePath) {
        var match = articlePath.match(/^\/专栏\/([^/]+)(?:\/.*)?$/);
        return match ? '/专栏/' + match[1] : '';
    }

    function isCourseDirectoryPath(articlePath) {
        return /^\/专栏\/[^/]+$/.test(articlePath);
    }

    function isCourseRootPath(articlePath) {
        return articlePath === '/专栏';
    }

    function isArticlePage() {
        return isArticlePath(getArticlePath());
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
        scrollY = Math.round(scrollY);
        return {
            articlePath: getArticlePath(),
            articleTitle: getArticleTitle(),
            progressPercent: progress,
            scrollY: scrollY,
            finished: progress === 100
        };
    }

    function mergeHighestProgress(payload) {
        var merged = Object.assign({}, payload);
        state.highestProgress = Math.max(state.highestProgress, merged.progressPercent);
        state.highestScrollY = Math.max(state.highestScrollY, merged.scrollY);
        merged.progressPercent = state.highestProgress;
        merged.scrollY = state.highestScrollY;
        merged.finished = merged.progressPercent === 100;
        return merged;
    }

    function loginPathForCurrentLocation() {
        var current = window.location.pathname || '/';
        current += window.location.search || '';
        current += window.location.hash || '';
        return '/login.html?next=' + encodeURIComponent(current);
    }

    function redirectToLogin() {
        window.location.replace(loginPathForCurrentLocation());
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

        document.getElementById('reading-progress-logout').addEventListener('click', function () {
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
            login.href = loginPathForCurrentLocation();
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
            return Promise.resolve();
        }
        return request('/reading-progress?articlePath=' + encodeURIComponent(getArticlePath())).then(function (data) {
            if (!data.found || !data.item) {
                return;
            }
            state.highestProgress = Math.max(state.highestProgress, data.item.progressPercent || 0);
            state.highestScrollY = Math.max(state.highestScrollY, data.item.scrollY || 0);
            if (!data.item.scrollY || data.item.progressPercent <= 0) {
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
        var payload = mergeHighestProgress(getProgress());
        if (!force) {
            if (now - state.lastSavedAt < SAVE_INTERVAL_MS) {
                return;
            }
            if (payload.progressPercent - state.lastSavedProgress < MIN_PROGRESS_DELTA) {
                return;
            }
        }
        state.lastSavedAt = now;
        state.lastSavedProgress = Math.max(state.lastSavedProgress, payload.progressPercent);
        request('/reading-progress', {
            method: 'PUT',
            keepalive: force,
            body: JSON.stringify(payload)
        }).catch(function () {});
    }

    function uniqueArticleLinks(container, coursePath, basePath) {
        var links = container ? container.querySelectorAll('a[href]') : [];
        var articles = [];
        var seen = {};
        for (var i = 0; i < links.length; i++) {
            var articlePath = pathFromLink(links[i], basePath);
            if (isArticlePath(articlePath) && articlePath.indexOf(coursePath + '/') === 0 && !seen[articlePath]) {
                seen[articlePath] = true;
                articles.push({ path: articlePath, link: links[i] });
            }
        }
        return articles;
    }

    function getCourseArticles(coursePath, pagePath) {
        var container = isCourseDirectoryPath(pagePath) ? document.querySelector('.book-post') : document.querySelector('.book-menu');
        return uniqueArticleLinks(container, coursePath);
    }

    function requestChapterProgress(articlePaths) {
        return request('/reading-progress/batch', {
            method: 'POST',
            body: JSON.stringify({ articlePaths: articlePaths })
        });
    }

    function progressMap(items) {
        var map = {};
        (items || []).forEach(function (item) {
            if (typeof item.progressPercent === 'number') {
                map[canonicalArticlePath(item.articlePath)] = item;
            }
        });
        return map;
    }

    function learningStatusClass(progress) {
        if (!progress) {
            return 'learning-status-unstarted';
        }
        return progress.progressPercent === 100 ? 'learning-status-completed' : 'learning-status-in-progress';
    }

    function courseSummaryStatusClass(summary) {
        var average = Number(summary && summary.averageProgressPercent) || 0;
        if (average >= 100) {
            return 'learning-status-completed';
        }
        return average > 0 ? 'learning-status-in-progress' : 'learning-status-unstarted';
    }

    function menuStatus(progress) {
        if (!progress) {
            return '未学习';
        }
        if (progress.progressPercent === 0) {
            return '已开始';
        }
        if (progress.progressPercent === 100) {
            return '完成';
        }
        return progress.progressPercent + '%';
    }

    function appendMenuChapterStatus(coursePath, records) {
        var menuLinks = document.querySelectorAll('.book-menu a.menu-item[href]');
        for (var i = 0; i < menuLinks.length; i++) {
            var articlePath = pathFromLink(menuLinks[i]);
            if (!isArticlePath(articlePath) || articlePath.indexOf(coursePath + '/') !== 0 || menuLinks[i].parentNode.querySelector('.course-menu-status')) {
                continue;
            }
            var status = document.createElement('span');
            status.className = 'course-menu-status ' + learningStatusClass(records[articlePath]);
            status.textContent = menuStatus(records[articlePath]);
            menuLinks[i].parentNode.appendChild(status);
        }
    }

    function requestCourseResumes(coursePaths) {
        return request('/reading-progress/course-resumes', {
            method: 'POST',
            body: JSON.stringify({ coursePaths: coursePaths })
        });
    }

    function findCourseResume(data, coursePath) {
        var items = data && data.items ? data.items : [];
        for (var i = 0; i < items.length; i++) {
            if (canonicalArticlePath(items[i].coursePath) === coursePath) {
                return items[i];
            }
        }
        return null;
    }

    function enhanceCourseDirectory(pagePath) {
        var coursePath = getCoursePath(pagePath);
        var articles = getCourseArticles(coursePath, pagePath);
        if (!articles.length) {
            return;
        }
        if (!state.user) {
            redirectToLogin();
            return;
        }
        requestCourseResumes([coursePath]).then(function (data) {
            var resume = findCourseResume(data, coursePath);
            var articlePath = resume ? canonicalArticlePath(resume.articlePath) : '';
            window.location.replace(isCourseArticlePath(articlePath, coursePath) ? articlePath : articles[0].path);
        }).catch(function () {});
    }

    function enhanceCourseArticle(pagePath) {
        var coursePath = getCoursePath(pagePath);
        var articles = getCourseArticles(coursePath, pagePath);
        if (!articles.length) {
            return;
        }
        requestChapterProgress(articles.map(function (article) { return article.path; })).then(function (data) {
            appendMenuChapterStatus(coursePath, progressMap(data.items));
        }).catch(function () {});
    }

    function courseSummaryText(summary) {
        var total = summary.articleCount || 0;
        var learned = summary.learnedArticleCount || 0;
        var average = summary.averageProgressPercent || 0;
        return learned ? '共 ' + total + ' 讲｜已学 ' + learned + ' 讲｜' + average + '%' : '共 ' + total + ' 讲｜未学习｜0%';
    }

    function appendCourseSummary(course, summary) {
        var item = course.link.parentNode;
        if (!item || item.querySelector('.course-root-summary')) {
            return;
        }
        var statusClass = courseSummaryStatusClass(summary);
        var details = document.createElement('span');
        details.className = 'course-root-summary ' + statusClass;
        details.textContent = courseSummaryText(summary);
        var progress = document.createElement('span');
        progress.className = 'course-root-progress ' + statusClass;
        var filled = document.createElement('span');
        filled.style.width = Math.max(0, Math.min(100, summary.averageProgressPercent || 0)) + '%';
        progress.appendChild(filled);
        item.appendChild(details);
        item.appendChild(progress);
    }

    function enhanceCourseRoot(pagePath) {
        if (!isCourseRootPath(pagePath)) {
            return;
        }
        var courseLinks = document.querySelectorAll('.book-post a[href]');
        var courses = [];
        var seen = {};
        for (var i = 0; i < courseLinks.length; i++) {
            var coursePath = pathFromLink(courseLinks[i]);
            if (isCourseDirectoryPath(coursePath) && !seen[coursePath]) {
                seen[coursePath] = true;
                courses.push({ path: coursePath, link: courseLinks[i] });
            }
        }
        if (!courses.length) {
            return;
        }
        request('/reading-progress/courses').then(function (data) {
            var summaries = {};
            (data.items || []).forEach(function (item) {
                summaries[canonicalArticlePath(item.coursePath)] = item;
            });
            courses.forEach(function (course) {
                if (summaries[course.path]) {
                    appendCourseSummary(course, summaries[course.path]);
                }
            });
        }).catch(function () {});
        requestCourseResumes(courses.map(function (course) { return course.path; })).then(function (data) {
            courses.forEach(function (course) {
                var resume = findCourseResume(data, course.path);
                var articlePath = resume ? canonicalArticlePath(resume.articlePath) : '';
                if (!isCourseArticlePath(articlePath, course.path)) {
                    return;
                }
                course.link.addEventListener('click', function (event) {
                    if (!normalPrimaryClick(event, course.link)) {
                        return;
                    }
                    event.preventDefault();
                    window.location.assign(articlePath);
                });
            });
        }).catch(function () {});
    }

    function isOtherRootPath(articlePath) {
        return articlePath === '/其他';
    }

    function getOtherCategoryPath(articlePath) {
        var match = articlePath.match(/^\/其他\/(恋爱必修课|文章|极客时间)(?:\/.*)?$/);
        return match ? '/其他/' + match[1] : '';
    }

    function isOtherCategoryDirectoryPath(articlePath) {
        return /^\/其他\/(?:恋爱必修课|文章|极客时间)$/.test(articlePath);
    }

    function getOtherCategoryArticles(categoryPath) {
        return uniqueArticleLinks(document.querySelector('.book-post'), categoryPath);
    }

    function fetchOtherCategoryArticles(categoryPath) {
        if (state.otherCategoryArticles[categoryPath]) {
            return state.otherCategoryArticles[categoryPath];
        }
        state.otherCategoryArticles[categoryPath] = fetch(categoryPath + '/', {
            credentials: 'same-origin',
            headers: { 'Accept': 'text/html' }
        }).then(function (resp) {
            if (!resp.ok) {
                throw new Error('分类目录加载失败');
            }
            return resp.text();
        }).then(function (html) {
            var doc = new DOMParser().parseFromString(html, 'text/html');
            return uniqueArticleLinks(doc.querySelector('.book-post'), categoryPath, window.location.origin + categoryPath + '/');
        });
        return state.otherCategoryArticles[categoryPath];
    }

    function latestReadArticle(articles, records) {
        var latest = null;
        var latestTime = -1;
        for (var i = 0; i < articles.length; i++) {
            var progress = records[articles[i].path];
            if (!progress) {
                continue;
            }
            var timestamp = Date.parse(progress.lastReadAt || '');
            timestamp = isNaN(timestamp) ? 0 : timestamp;
            if (!latest || timestamp > latestTime) {
                latest = articles[i];
                latestTime = timestamp;
            }
        }
        return latest;
    }

    function resolveOtherCategoryTarget(articles) {
        if (!articles.length) {
            return Promise.reject(new Error('分类中没有文章'));
        }
        if (!state.user) {
            redirectToLogin();
            return Promise.resolve('');
        }
        return requestChapterProgress(articles.map(function (article) { return article.path; })).then(function (data) {
            var latest = latestReadArticle(articles, progressMap(data.items));
            return latest ? latest.path : articles[0].path;
        });
    }

    function replaceOtherArticleMenu(categoryPath, articles) {
        var menu = document.querySelector('.book-menu');
        if (!menu || menu.querySelector('.other-article-navigation')) {
            return null;
        }
        var navigation = document.createElement('div');
        navigation.className = 'other-article-navigation';
        var back = document.createElement('a');
        back.className = 'other-menu-back menu-item';
        back.href = '/其他/';
        back.textContent = '← 返回其他';
        navigation.appendChild(back);
        var list = document.createElement('ul');
        for (var i = 0; i < articles.length; i++) {
            var item = document.createElement('li');
            var link = document.createElement('a');
            link.className = 'menu-item';
            link.href = articles[i].path;
            link.textContent = articles[i].link.textContent.trim() || articles[i].path;
            item.appendChild(link);
            list.appendChild(item);
        }
        navigation.appendChild(list);
        menu.textContent = '';
        menu.appendChild(navigation);
        return navigation;
    }

    function appendOtherMenuStatuses(navigation, articles, records) {
        if (!navigation) {
            return;
        }
        var links = navigation.querySelectorAll('a.menu-item[href]');
        for (var i = 0; i < links.length; i++) {
            var articlePath = pathFromLink(links[i]);
            if (!isOtherArticlePath(articlePath) || links[i].parentNode.querySelector('.other-menu-status')) {
                continue;
            }
            var status = document.createElement('span');
            status.className = 'other-menu-status ' + learningStatusClass(records[articlePath]);
            status.textContent = menuStatus(records[articlePath]);
            links[i].parentNode.appendChild(status);
        }
    }

    function enhanceOtherCategoryDirectory(pagePath) {
        var categoryPath = getOtherCategoryPath(pagePath);
        var articles = getOtherCategoryArticles(categoryPath);
        if (!articles.length) {
            return;
        }
        resolveOtherCategoryTarget(articles).then(function (target) {
            if (target) {
                window.location.replace(target);
            }
        }).catch(function () {});
    }

    function enhanceOtherArticle(pagePath) {
        var categoryPath = getOtherCategoryPath(pagePath);
        if (!categoryPath) {
            return;
        }
        fetchOtherCategoryArticles(categoryPath).then(function (articles) {
            if (!articles.length) {
                return;
            }
            var navigation = replaceOtherArticleMenu(categoryPath, articles);
            if (!state.user) {
                return;
            }
            requestChapterProgress(articles.map(function (article) { return article.path; })).then(function (data) {
                appendOtherMenuStatuses(navigation, articles, progressMap(data.items));
            }).catch(function () {});
        }).catch(function () {});
    }

    function normalPrimaryClick(event, link) {
        return event.button === 0 && !event.ctrlKey && !event.metaKey && !event.shiftKey && !event.altKey && link.target !== '_blank';
    }

    function enhanceOtherRoot(pagePath) {
        if (!isOtherRootPath(pagePath)) {
            return;
        }
        var links = document.querySelectorAll('.book-post a[href]');
        for (var i = 0; i < links.length; i++) {
            (function (link) {
                var categoryPath = pathFromLink(link);
                if (!isOtherCategoryDirectoryPath(categoryPath)) {
                    return;
                }
                link.addEventListener('click', function (event) {
                    if (!normalPrimaryClick(event, link)) {
                        return;
                    }
                    event.preventDefault();
                    if (!state.user) {
                        redirectToLogin();
                        return;
                    }
                    fetchOtherCategoryArticles(categoryPath).then(function (articles) {
                        return resolveOtherCategoryTarget(articles);
                    }).then(function (target) {
                        if (target) {
                            window.location.assign(target);
                        }
                    }).catch(function () {
                        if (state.user) {
                            window.location.assign(categoryPath);
                        }
                    });
                });
            })(links[i]);
        }
    }

    function enhanceCoursePages() {
        var pagePath = getArticlePath();
        if (isCourseRootPath(pagePath)) {
            if (state.user) {
                enhanceCourseRoot(pagePath);
            }
        } else if (isCourseDirectoryPath(pagePath)) {
            enhanceCourseDirectory(pagePath);
        } else if (isCourseArticlePath(pagePath) && state.user) {
            enhanceCourseArticle(pagePath);
        }
    }

    function enhanceOtherPages() {
        var pagePath = getArticlePath();
        if (isOtherRootPath(pagePath)) {
            enhanceOtherRoot(pagePath);
        } else if (isOtherCategoryDirectoryPath(pagePath)) {
            enhanceOtherCategoryDirectory(pagePath);
        } else if (isOtherArticlePath(pagePath)) {
            enhanceOtherArticle(pagePath);
        }
    }

    ready(function () {
        buildPanel();
        loadCurrentUser().then(function () {
            enhanceCoursePages();
            enhanceOtherPages();
            return maybeRestore();
        }).then(function () {
            if (state.user && isArticlePage()) {
                scheduleSave(true);
                window.addEventListener('scroll', function () { scheduleSave(false); }, { passive: true });
                window.addEventListener('pagehide', function () { saveProgress(true); });
                window.addEventListener('beforeunload', function () { saveProgress(true); });
            }
        });
    });
})();
