import { $, $$ } from '../utils/dom.js';
import { apiRequest, buildQueryString } from '../utils/api.js';

/**
 * 告警中心管理器
 */
export class AlertManager {
    constructor() {
        this.page = 0;
        this.pageSize = 50;
        this.total = 0;
        this.selectedAlerts = new Set();
    }

    async listAlerts() {
        try {
            const filter = {
                limit: this.pageSize,
                offset: this.page * this.pageSize
            };

            const status = $('#alertFilterStatus')?.value;
            if (status) filter.status = status;

            const severity = $('#alertFilterSeverity')?.value;
            if (severity) filter.severity = severity;

            const type = $('#alertFilterType')?.value;
            if (type) filter.type = type;

            const queryString = buildQueryString(filter);
            const result = await apiRequest(`/alerts/list?${queryString}`);
            this.total = Number(result?.total) || 0;
            this.renderSummary(result?.summary || {});
            this.renderAlerts(Array.isArray(result?.data) ? result.data : []);
            this.updatePagination();
        } catch (error) {
            app.logger.error('加载告警列表失败: ' + error);
        }
    }

    async scanAlerts() {
        try {
            await apiRequest('/alerts/scan', 'POST', {});
            app.logger.success('告警状态扫描完成');
            this.resetAlertPageAndList();
        } catch (error) {
            app.logger.error('告警状态扫描失败: ' + error);
        }
    }

    renderSummary(summary) {
        $('#alertSummaryActive').textContent = Number(summary.active) || 0;
        $('#alertSummaryCritical').textContent = Number(summary.critical) || 0;
        $('#alertSummaryWarning').textContent = Number(summary.warning) || 0;
    }

    renderAlerts(alerts) {
        const tbody = $('#alertList');
        if (!tbody) return;

        this.selectedAlerts.clear();
        const checkAll = $('#alertCheckAll');
        if (checkAll) checkAll.checked = false;

        if (!alerts || alerts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="9" class="empty-table-cell">暂无告警</td></tr>';
            return;
        }

        tbody.innerHTML = alerts.map(alert => {
            const active = alert.status === 'active';
            return `
                <tr>
                    <td><input type="checkbox" value="${alert.id}" ${active ? '' : 'disabled'} onchange="app.alertManager.toggleAlertSelection(${alert.id})"></td>
                    <td><span class="alert-badge is-${this.escapeHtml(alert.severity)}">${this.formatSeverity(alert.severity)}</span></td>
                    <td>${this.formatType(alert.type)}</td>
                    <td>${this.escapeHtml(alert.source || '-')}</td>
                    <td>
                        <span class="alert-message">${this.escapeHtml(alert.message || '-')}</span>
                        <span class="alert-detail">${this.escapeHtml(alert.detail || '')}</span>
                    </td>
                    <td>${alert.count || 1}</td>
                    <td>${new Date(alert.last_seen).toLocaleString()}</td>
                    <td><span class="alert-status ${active ? '' : 'is-resolved'}">${active ? '活跃' : '已解决'}</span></td>
                    <td>
                        ${active ? `<button class="btn btn-small btn-secondary" onclick="app.alertManager.resolveAlert(${alert.id})">解决</button>` : '-'}
                    </td>
                </tr>
            `;
        }).join('');
    }

    toggleAlertSelection(id) {
        if (this.selectedAlerts.has(id)) {
            this.selectedAlerts.delete(id);
        } else {
            this.selectedAlerts.add(id);
        }
    }

    toggleCheckAll() {
        const checkAll = $('#alertCheckAll');
        if (!checkAll) return;

        const checkboxes = $$('#alertList input[type="checkbox"]:not(:disabled)');
        checkboxes.forEach(checkbox => {
            checkbox.checked = checkAll.checked;
            const id = parseInt(checkbox.value);
            if (checkAll.checked) {
                this.selectedAlerts.add(id);
            } else {
                this.selectedAlerts.delete(id);
            }
        });
    }

    async resolveAlert(id) {
        try {
            await apiRequest('/alerts/resolve', 'POST', { ids: [id] });
            app.logger.success('告警已解决');
            this.listAlerts();
        } catch (error) {
            app.logger.error('解决告警失败: ' + error);
        }
    }

    async resolveSelectedAlerts() {
        if (this.selectedAlerts.size === 0) {
            app.logger.error('请先选择要解决的告警');
            return;
        }

        try {
            await apiRequest('/alerts/resolve', 'POST', { ids: Array.from(this.selectedAlerts) });
            app.logger.success('选中告警已解决');
            this.selectedAlerts.clear();
            this.listAlerts();
        } catch (error) {
            app.logger.error('解决选中告警失败: ' + error);
        }
    }

    async resolveAllAlerts() {
        if (!confirm('确定要解决全部活跃告警吗？')) {
            return;
        }

        try {
            await apiRequest('/alerts/resolve', 'POST', { all: true });
            app.logger.success('全部活跃告警已解决');
            this.selectedAlerts.clear();
            this.listAlerts();
        } catch (error) {
            app.logger.error('解决全部告警失败: ' + error);
        }
    }

    resetAlertPageAndList() {
        this.page = 0;
        this.listAlerts();
    }

    prevPage() {
        if (this.page > 0) {
            this.page--;
            this.listAlerts();
        }
    }

    nextPage() {
        const totalPages = Math.ceil(this.total / this.pageSize);
        if (this.page < totalPages - 1) {
            this.page++;
            this.listAlerts();
        }
    }

    updatePagination() {
        const totalPages = Math.max(1, Math.ceil(this.total / this.pageSize));
        $('#alertPageInfo').textContent = `第 ${this.page + 1} 页 / 共 ${totalPages} 页 (总计: ${this.total} 条)`;
        $('#alertPrevPageBtn').disabled = this.page === 0;
        $('#alertNextPageBtn').disabled = this.page >= totalPages - 1;
    }

    formatSeverity(severity) {
        const labels = {
            critical: '严重',
            warning: '警告',
            info: '信息'
        };
        return labels[severity] || severity || '-';
    }

    formatType(type) {
        const labels = {
            no_modem: '无可用卡',
            modem_offline: '卡离线',
            sim_status_unknown: 'SIM 状态未知',
            sim_not_ready: 'SIM 未就绪',
            network_status_unknown: '网络状态未知',
            network_not_registered: '网络未注册',
            signal_unknown: '信号状态未知',
            no_signal: '无信号',
            low_signal: '信号弱',
            sms_send_failed: '短信发送失败',
            webhook_failed: 'Webhook 失败'
        };
        return this.escapeHtml(labels[type] || type || '-');
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = String(text ?? '');
        return div.innerHTML;
    }
}
