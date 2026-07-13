var path = window.location.pathname
var cookie = getCookie("lastPath");
console.log(path)
if (path.replace("/", "") === "") {
    if (cookie.replace("/", "") !== "") {
        console.log(cookie)
        document.getElementById("tip").innerHTML = "<a href='" + cookie + "'>跳转到上次进度</a>"
    }
} else {
    setCookie("lastPath", path)
}

window.onload = function () {
    var titleEle = document.getElementById("title");
    if (!titleEle) {
        return
    }
    var title = titleEle.getAttribute("data-id")
    console.log("title=" + title)
    var eleList = document.getElementsByClassName("menu-item")
    for (var i = 0; i < eleList.length; i++) {  //遍历数组
        console.log("eleText=" + eleList[i].id)
        if (eleList[i].id.startsWith(title)) {
            eleList[i].classList.add("current-tab")
            if (i > 0) {
                document.getElementById("prePage").innerHTML = "<a href='" + eleList[i - 1].getAttribute("href") + "'>上一页</a>"
            }
            if (i < eleList.length) {
                document.getElementById("nextPage").innerHTML = "<a href='" + eleList[i + 1].getAttribute("href") + "'>下一页</a>"
            }

        }
    }
}

function setCookie(cname, cvalue) {
    var d = new Date();
    d.setTime(d.getTime() + (180 * 24 * 60 * 60 * 1000));
    var expires = "expires=" + d.toGMTString();
    document.cookie = cname + "=" + cvalue + "; " + expires + ";path = /";
}

function getCookie(cname) {
    var name = cname + "=";
    var ca = document.cookie.split(';');
    for (var i = 0; i < ca.length; i++) {
        var c = ca[i].trim();
        if (c.indexOf(name) === 0) return c.substring(name.length, c.length);
    }
    return "";
}

hljs.initHighlightingOnLoad()

function add_inner() {
    let inner = document.querySelector('.sidebar-toggle-inner')
    inner.classList.add('show')
}

function remove_inner() {
    let inner = document.querySelector('.sidebar-toggle-inner')
    inner.classList.remove('show')
}

function sidebar_toggle() {
    let sidebar_toggle = document.querySelector('.sidebar-toggle')
    let sidebar = document.querySelector('.book-sidebar')
    let content = document.querySelector('.off-canvas-content')
    if (sidebar_toggle.classList.contains('extend')) { // show
        sidebar_toggle.classList.remove('extend')
        sidebar.classList.remove('hide')
        content.classList.remove('extend')
    } else { // hide
        sidebar_toggle.classList.add('extend')
        sidebar.classList.add('hide')
        content.classList.add('extend')
    }
}


function open_sidebar() {
    let sidebar = document.querySelector('.book-sidebar')
    let overlay = document.querySelector('.off-canvas-overlay')
    sidebar.classList.add('show')
    overlay.classList.add('show')
}
function hide_canvas() {
    let sidebar = document.querySelector('.book-sidebar')
    let overlay = document.querySelector('.off-canvas-overlay')
    sidebar.classList.remove('show')
    overlay.classList.remove('show')
}

// 全站右上角悬浮GitHub图标
(function() {
  var githubDiv = document.createElement('div');
  githubDiv.style.position = 'fixed';
  githubDiv.style.top = '24px';
  githubDiv.style.right = '24px';
  githubDiv.style.zIndex = '9999';
  githubDiv.style.cursor = 'pointer';
  githubDiv.title = '访问 GitHub 仓库';

  var githubLink = document.createElement('a');
  githubLink.href = 'https://github.com/xixiwenxuanhe/learn-liangliang';
  githubLink.target = '_blank';
  githubLink.rel = 'noopener noreferrer';

  // 使用本地SVG图片
  var githubImg = document.createElement('img');
  githubImg.src = '/img/github.svg';
  githubImg.alt = 'GitHub';
  githubImg.style.width = '40px';
  githubImg.style.height = '40px';
  githubImg.style.display = 'block';

  githubLink.appendChild(githubImg);
  githubDiv.appendChild(githubLink);
  document.body.appendChild(githubDiv);
})();

// 移除"因收到Google相关通知，网站将会择期关闭"提示
(function() {
  var allDivs = document.getElementsByTagName('div');
  for (var i = 0; i < allDivs.length; i++) {
    var div = allDivs[i];
    if (
      div.getAttribute('align') === 'center' &&
      div.innerText &&
      div.innerText.indexOf('因收到Google相关通知，网站将会择期关闭') !== -1 &&
      div.querySelector('a[href*="lumendatabase.org/notices/44265620"]')
    ) {
      div.parentNode.removeChild(div);
      break; // 只移除第一个找到的即可
    }
  }
})();

// 动态加载阅读进度同步脚本
(function() {
    var script = document.createElement('script');
    script.src = '/static/reading-progress.js';
    script.defer = true;
    document.body.appendChild(script);
})();

// 修改页脚内容
(function() {
    // 查找所有包含版权信息的段落
    var paragraphs = document.getElementsByTagName('p');
    for (var i = 0; i < paragraphs.length; i++) {
        var p = paragraphs[i];
        if (p.innerHTML.includes('© 2019 - 2023') && p.innerHTML.includes('Liangliang Lee')) {
            // 设置艺术字体样式
            p.style.fontFamily = "'Brush Script MT', cursive";
            p.style.fontSize = '1.2em';
            p.style.color = '#b8860b';
            p.style.textAlign = 'center';
            p.style.margin = '2rem auto';
            p.style.padding = '1rem';
            p.style.borderTop = '1px solid #b8860b';
            p.style.borderBottom = '1px solid #b8860b';
            p.style.letterSpacing = '1px';
            
            // 设置新的内容，添加符号
            p.innerHTML = '内容源自网络收集，仅供交流学习使用<br>✦ xixiwenxuanhe 2025-2026 ✦';
            break; // 找到并修改后退出循环
        }
    }
})();

