/**
 * DOM 选择器快捷方式
 * 提供便捷的DOM元素选择方法
 */

/** 选择单个元素 */
export const $ = document.querySelector.bind(document);

/** 选择多个元素 */
export const $$ = document.querySelectorAll.bind(document);

/**
 * 转义HTML特殊字符
 * 防止XSS攻击，将特殊字符转换为HTML实体
 * @param {string} text - 要转义的文本
 * @returns {string} 转义后的文本
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * UI渲染器类
 * 负责模板渲染和UI更新
 */
export class UIrender {
    /**
     * 构造函数
     * @param {Object} templates - 模板对象
     */
    constructor() {
        this.templates = {};
        this.extract('modemInfo', 'modemInfo');
        this.extract('signalInfo', 'signalInfo');
        this.extract('smsList', 'smsItem');
        this.extract('smsdbList', 'smsdbItem');
        this.extract('webhookList', 'webhookItem');
    }

    /**
     * 从DOM中提取模板
     * @param {string} elementId - 元素ID
     * @param {string} templateKey - 模板键名
     */
    extract(elementId, templateKey) {
        if (!this.templates[templateKey]) {
            const element = $(`#${elementId}`);
            if (element) {
                this.templates[templateKey] = element.innerHTML || '';
                element.innerHTML = this.loadingMarkup(element, elementId);
            }
        }
    }

    loadingMarkup(element, elementId) {
        const loading = '<div class="loading-state"><span class="loading-spinner"></span><span>加载中</span></div>';
        if (elementId === 'modemInfo') {
            return this.infoSkeleton(16);
        }
        if (elementId === 'signalInfo') {
            return this.infoSkeleton(18);
        }
        if (element.tagName === 'TBODY') {
            const colspan = elementId === 'webhookList' ? 7 : 8;
            return `<tr><td colspan="${colspan}">${loading}</td></tr>`;
        }
        return loading;
    }

    infoSkeleton(count) {
        return Array.from({ length: count }, () => `
            <div class="info-item info-skeleton">
                <span class="skeleton-line skeleton-label"></span>
                <span class="skeleton-line skeleton-value"></span>
            </div>
        `).join('');
    }

    /**
     * 渲染模板
     * @param {string} templateKey - 模板键名
     * @param {Object} data - 模板数据
     * @returns {string} 渲染后的HTML
     */
    render(templateKey, data) {
        const template = this.templates[templateKey] || '';
        return template.replace(/\{([\w.]+)\}/g, (_, path) => {
            const value = path.split('.').reduce((obj, k) => (obj && obj[k] !== undefined ? obj[k] : undefined), data);
            const safe = value === undefined || value === null || value === '' ? '-' : value;
            return typeof safe === 'object' ? JSON.stringify(safe) : escapeHtml(String(safe));
        });
    }
}
