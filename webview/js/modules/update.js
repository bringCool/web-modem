import { $ } from '../utils/dom.js';
import { apiRequest } from '../utils/api.js';

/**
 * 系统更新管理器
 */
export class UpdateManager {
    constructor() {
        this.lastResult = null;
    }

    async checkUpdate() {
        try {
            app.logger.info('正在检查系统更新...');
            const result = await apiRequest('/update/check');
            this.lastResult = result;
            this.renderUpdateResult(result);

            if (result.has_update) {
                app.logger.success(`发现新版本: ${result.latest_version}`);
            } else {
                app.logger.info('当前已是最新版本');
            }
            return result;
        } catch (error) {
            this.renderUpdateError(error);
            app.logger.error('检查系统更新失败: ' + error);
            throw error;
        }
    }

    async applyUpdate() {
        const result = this.lastResult || await this.checkUpdate();
        if (!result.has_update) {
            app.logger.info('当前已是最新版本');
            return;
        }

        if (!confirm(`确定更新到 ${result.latest_version} 并重启应用吗？`)) {
            return;
        }

        try {
            this.setStatus('正在下载并应用更新...');
            const response = await apiRequest('/update/apply', 'POST', { restart: true });
            this.setStatus('更新完成，正在重启...');
            app.logger.success(`系统已更新到 ${response.result?.latest_version || result.latest_version}`);
        } catch (error) {
            this.renderUpdateError(error);
            app.logger.error('系统更新失败: ' + error);
        }
    }

    renderUpdateResult(result) {
        $('#updateCurrentVersion').textContent = result.current_version || '-';
        $('#updateLatestVersion').textContent = result.latest_version || '-';
        this.setStatus(result.has_update ? '发现新版本' : '已是最新版本');

        const release = $('#updateRelease');
        if (!release) return;
        release.textContent = result.info?.release || '';
        release.classList.toggle('is-empty', !release.textContent);
    }

    renderUpdateError(error) {
        this.setStatus('检查失败');
        const release = $('#updateRelease');
        if (release) {
            release.textContent = String(error);
            release.classList.remove('is-empty');
        }
    }

    setStatus(text) {
        const status = $('#updateStatus');
        if (status) {
            status.textContent = text;
        }
    }
}
