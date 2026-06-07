import { $ } from '../utils/dom.js';
import { apiRequest, buildQueryString } from '../utils/api.js';
import { PRESET_TEMPLATES } from './webhook.tpl.js';

/**
 * Webhook管理器类
 * 负责管理Webhook配置，包括创建、编辑、删除、测试等功能
 */
export class WebhookManager {
    /**
     * 构造函数
     * 初始化Webhook管理器的基本状态和属性
     */
    constructor() {
        // 当前编辑的 Webhook ID
        this.currentWebhookId = null;
        this.deliveryPage = 0;
        this.deliveryPageSize = 50;
        this.deliveryTotal = 0;
        // 初始化预设模板选项
        this.initPresetTemplates();
        this.updateMatchInputHint();
    }

    /**
     * 初始化预设模板下拉选项
     * 根据 PRESET_TEMPLATES 自动生成选项
     */
    initPresetTemplates() {
        const select = $('#webhookTemplateSelect');
        if (!select) return;

        // 清空现有选项（保留第一个"自定义"选项）
        const customOption = select.querySelector('option[value="custom"]');
        select.innerHTML = '';
        if (customOption) {
            select.appendChild(customOption);
        } else {
            const newCustomOption = document.createElement('option');
            newCustomOption.value = 'custom';
            newCustomOption.textContent = '自定义';
            select.appendChild(newCustomOption);
        }

        // 根据 PRESET_TEMPLATES 生成选项
        Object.keys(PRESET_TEMPLATES).forEach(key => {
            const preset = PRESET_TEMPLATES[key];
            if (preset.name && preset.template) {
                const option = document.createElement('option');
                option.value = key;
                option.textContent = preset.name;
                select.appendChild(option);
            }
        });
    }

    /* =========================================
       Webhook管理 (Webhook Management)
       ========================================= */

    /**
     * 列出Webhook配置
     * 获取所有已配置的Webhook列表
     */
    async listWebhooks() {
        try {
            const result = await apiRequest('/webhook/list');
            const webhooks = Array.isArray(result) ? result : [];
            const tbody = $('#webhookList');
            if (!webhooks || webhooks.length === 0) {
                tbody.innerHTML = '<tr><td colspan="7" class="empty-table-cell">暂无 Webhook 配置</td></tr>';
                this.populateDeliveryWebhookFilter([]);
                this.listDeliveries();
                return;
            }
            tbody.innerHTML = webhooks.map(webhook => app.render.render('webhookItem', {
                id: webhook.id,
                name: webhook.name,
                url: webhook.url,
                event: this.formatEventType(webhook.event_type),
                match: this.formatMatch(webhook.match_type, webhook.match_value),
                status_label: webhook.enabled ? '启用' : '停用',
                status_class: webhook.enabled ? 'is-enabled' : 'is-disabled',
                created_at: new Date(webhook.created_at).toLocaleString()
            })).join('');
            this.populateDeliveryWebhookFilter(webhooks);
            this.listDeliveries();
        } catch (error) {
            app.logger.error('加载 Webhook 列表失败: ' + error);
        }
    }

    async editWebhook(id) {
        try {
            const queryString = buildQueryString({ id });
            const webhook = await apiRequest(`/webhook/get?${queryString}`);
            this.currentWebhookId = id;
            $('#webhookFormTitle').textContent = '编辑 Webhook';
            $('#webhookName').value = webhook.name;
            $('#webhookURL').value = webhook.url;
            $('#webhookTemplate').value = webhook.template;
            $('#webhookEnabledCheckbox').checked = webhook.enabled;
            $('#webhookEventType').value = webhook.event_type || 'sms_received';
            $('#webhookMatchType').value = webhook.match_type || 'all';
            $('#webhookMatchValue').value = webhook.match_value || '';
            $('#webhookRetryCount').value = Number.isInteger(webhook.retry_count) ? webhook.retry_count : 2;
            $('#webhookRetryInterval').value = webhook.retry_interval_seconds || 2;
            $('#webhookRetryBackoff').checked = webhook.retry_backoff === true;
            $('#webhookTemplateSelect').value = 'custom';
            this.updateMatchInputHint();
            this.previewWebhook({ silent: true });
        } catch (error) {
            app.logger.error('加载 Webhook 详情失败: ' + error);
        }
    }

    resetForm() {
        this.currentWebhookId = null;
        $('#webhookFormTitle').textContent = '创建 Webhook';
        $('#webhookName').value = '';
        $('#webhookURL').value = '';
        $('#webhookTemplate').value = '{}';
        $('#webhookEnabledCheckbox').checked = true;
        $('#webhookEventType').value = 'sms_received';
        $('#webhookMatchType').value = 'all';
        $('#webhookMatchValue').value = '';
        $('#webhookRetryCount').value = '2';
        $('#webhookRetryInterval').value = '2';
        $('#webhookRetryBackoff').checked = true;
        $('#webhookTemplateSelect').value = 'custom';
        $('#webhookPreview').textContent = '{}';
        $('#webhookPreviewMeta').textContent = '模拟短信';
        this.updateMatchInputHint();
    }

    formatEventType(type) {
        const labels = {
            all: '全部事件',
            sms_received: '收到短信',
            sms_sent_success: '发送成功',
            sms_sent_failed: '发送失败',
            alert_triggered: '告警触发',
            alert_resolved: '告警恢复',
            sms_sync_completed: '同步完成',
            system_update_completed: '更新完成'
        };
        return labels[type || 'sms_received'] || type || '收到短信';
    }

    formatMatch(type, value) {
        const labels = {
            all: '全部匹配',
            receive_number: '接收号码',
            device_imei: 'IMEI',
            sim_iccid: 'ICCID',
            sim_imsi: 'IMSI',
            operator: '运营商',
            send_number: '发送号码',
            modem_name: '串口',
            direction: '方向',
            content_contains: '内容包含',
            alert_type: '告警类型',
            alert_severity: '告警级别',
            alert_source: '告警来源'
        };
        if (!type || type === 'all') return '全部匹配';
        return `${labels[type] || type}: ${value || '-'}`;
    }

    updateMatchInputHint() {
        const matchType = $('#webhookMatchType')?.value || 'all';
        const input = $('#webhookMatchValue');
        if (!input) return;

        const placeholders = {
            all: '全部匹配可留空',
            receive_number: '+8613800138000',
            send_number: '+8613800138001',
            direction: 'in 或 out',
            content_contains: '验证码',
            device_imei: '模块 IMEI',
            sim_iccid: '8986 开头的 ICCID',
            sim_imsi: '460 开头的 IMSI',
            operator: '运营商名称或 PLMN',
            modem_name: 'ttyUSB2',
            alert_type: 'low_signal',
            alert_severity: 'critical / warning / info',
            alert_source: 'ttyUSB2 或 system'
        };
        input.placeholder = placeholders[matchType] || '条件值';
        input.disabled = matchType === 'all';
        if (matchType === 'all') {
            input.value = '';
        }
    }

    /**
     * 应用预设模板
     * 当用户从下拉框选择预设模板时，自动填充模板内容
     */
    applyPresetTemplate() {
        const templateTextarea = $('#webhookTemplate');
        if (!templateTextarea) {
            return;
        }

        // 如果选择了自定义模板，不进行任何操作
        const select = $('#webhookTemplateSelect');
        const templateKey = select.value;
        if (templateKey === 'custom') {
            return;
        }

        // 获取预设模板
        const preset = PRESET_TEMPLATES[templateKey];
        if (preset && preset.template) {
            // 将预设模板格式化为JSON字符串，美化输出
            templateTextarea.value = JSON.stringify(preset.template, null, 2);
            this.previewWebhook({ silent: true });
        }
    }

    collectWebhookForm(options = {}) {
        const { requireName = true, requireURL = true } = options;
        const name = $('#webhookName').value.trim();
        const url = $('#webhookURL').value.trim();
        const template = $('#webhookTemplate').value.trim() || '{}';
        const enabled = $('#webhookEnabledCheckbox').checked;
        const eventType = $('#webhookEventType').value || 'sms_received';
        const matchType = $('#webhookMatchType').value;
        const matchValue = $('#webhookMatchValue').value.trim();
        const retryCount = Number.parseInt($('#webhookRetryCount').value, 10);
        const retryInterval = Number.parseInt($('#webhookRetryInterval').value, 10);
        const retryBackoff = $('#webhookRetryBackoff').checked;

        if (requireName && !name) {
            app.logger.error('请填写 Webhook 名称');
            return null;
        }

        if (requireURL && !url) {
            app.logger.error('请填写 Webhook URL');
            return null;
        }

        if (matchType !== 'all' && !matchValue) {
            app.logger.error('请填写 Webhook 条件值');
            return null;
        }

        if (!Number.isInteger(retryCount) || retryCount < 0 || retryCount > 10) {
            app.logger.error('重试次数必须是 0 到 10 之间的整数');
            return null;
        }

        if (!Number.isInteger(retryInterval) || retryInterval < 1 || retryInterval > 3600) {
            app.logger.error('重试间隔必须是 1 到 3600 秒之间的整数');
            return null;
        }

        if (!this.validateTemplate(template)) {
            return null;
        }

        return {
            name,
            url,
            template,
            enabled,
            event_type: eventType,
            match_type: matchType,
            match_value: matchType === 'all' ? '' : matchValue,
            retry_count: retryCount,
            retry_interval_seconds: retryInterval,
            retry_backoff: retryBackoff
        };
    }

    validateTemplate(template) {
        if (template && template !== '{}') {
            try {
                JSON.parse(template);
            } catch (e) {
                app.logger.error('模板必须是有效的 JSON 格式');
                return false;
            }
        }

        return true;
    }

    /**
     * 保存Webhook配置
     * 创建或更新Webhook设置
     */
    async saveWebhook() {
        const webhookData = this.collectWebhookForm();
        if (!webhookData) return;

        try {
            const previewOK = await this.previewWebhook({ silent: true, data: webhookData });
            if (!previewOK) return;

            if (this.currentWebhookId) {
                const queryString = buildQueryString({ id: this.currentWebhookId });
                await apiRequest(`/webhook/update?${queryString}`, 'PUT', webhookData);
                app.logger.success('Webhook 更新成功');
            } else {
                await apiRequest('/webhook', 'POST', webhookData);
                app.logger.success('Webhook 创建成功');
            }

            this.resetForm();
            this.listWebhooks();
        } catch (error) {
            app.logger.error('保存 Webhook 失败: ' + error);
        }
    }

    async deleteWebhook(id) {
        if (!confirm('确定要删除这个 Webhook 吗？')) {
            return;
        }

        try {
            const queryString = buildQueryString({ id });
            await apiRequest(`/webhook/delete?${queryString}`, 'DELETE');
            app.logger.success('Webhook 删除成功');
            this.listWebhooks();
        } catch (error) {
            app.logger.error('删除 Webhook 失败: ' + error);
        }
    }

    async previewWebhook(options = {}) {
        const data = options.data || this.collectWebhookForm({ requireName: false, requireURL: false });
        if (!data) return false;

        try {
            const result = await apiRequest('/webhook/preview', 'POST', data);
            $('#webhookPreview').textContent = result?.body || '{}';
            $('#webhookPreviewMeta').textContent = `${this.formatEventType(data.event_type)} · ${new Date().toLocaleString()}`;
            if (!options.silent) {
                app.logger.success('Webhook 模板预览已生成');
            }
            return true;
        } catch (error) {
            $('#webhookPreview').textContent = String(error?.message || error);
            $('#webhookPreviewMeta').textContent = '预览失败';
            if (!options.silent) {
                app.logger.error('Webhook 模板预览失败: ' + error);
            }
            return false;
        }
    }

    async testWebhook(id = null) {
        let requestSent = false;
        try {
            if (id) {
                // 测试已存在的webhook
                const queryString = buildQueryString({ id });
                requestSent = true;
                await apiRequest(`/webhook/test?${queryString}`, 'POST');
            } else {
                const webhookData = this.collectWebhookForm({ requireName: false, requireURL: true });
                if (!webhookData) return;
                webhookData.name = webhookData.name || '测试';
                webhookData.enabled = true;

                requestSent = true;
                await apiRequest('/webhook/test', 'POST', webhookData);
            }

            app.logger.success('Webhook 测试请求已发送');
        } catch (error) {
            app.logger.error('Webhook 测试失败: ' + error);
        } finally {
            if (requestSent) {
                this.resetDeliveryPageAndList();
            }
        }
    }

    populateDeliveryWebhookFilter(webhooks) {
        const select = $('#webhookDeliveryFilterWebhook');
        if (!select) return;

        const currentValue = select.value;
        select.innerHTML = '<option value="">全部 Webhook</option>';
        webhooks.forEach(webhook => {
            const option = document.createElement('option');
            option.value = String(webhook.id);
            option.textContent = webhook.name;
            select.appendChild(option);
        });
        select.value = Array.from(select.options).some(option => option.value === currentValue) ? currentValue : '';
    }

    async listDeliveries() {
        const tbody = $('#webhookDeliveryList');
        if (!tbody) return;

        try {
            const filter = {
                limit: this.deliveryPageSize,
                offset: this.deliveryPage * this.deliveryPageSize
            };

            const webhookID = $('#webhookDeliveryFilterWebhook')?.value;
            if (webhookID) {
                filter.webhook_id = webhookID;
            }

            const status = $('#webhookDeliveryFilterStatus')?.value;
            if (status) {
                filter.status = status;
            }

            const eventType = $('#webhookDeliveryFilterEvent')?.value;
            if (eventType) {
                filter.event_type = eventType;
            }

            const startDate = $('#webhookDeliveryFilterStartDate')?.value;
            if (startDate) {
                filter.start_time = new Date(startDate).toISOString();
            }

            const endDate = $('#webhookDeliveryFilterEndDate')?.value;
            if (endDate) {
                const end = new Date(endDate);
                end.setHours(23, 59, 59, 999);
                filter.end_time = end.toISOString();
            }

            const queryString = buildQueryString(filter);
            const result = await apiRequest(`/webhook/deliveries?${queryString}`);
            this.deliveryTotal = Number(result?.total) || 0;
            this.displayDeliveries(Array.isArray(result?.data) ? result.data : []);
            this.updateDeliveryPagination();
        } catch (error) {
            app.logger.error('加载 Webhook 发送记录失败: ' + error);
        }
    }

    displayDeliveries(deliveries) {
        const tbody = $('#webhookDeliveryList');
        if (!tbody) return;

        if (!deliveries || deliveries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="9" class="empty-table-cell">暂无发送记录</td></tr>';
            return;
        }

        tbody.innerHTML = deliveries.map(delivery => {
            const success = delivery.status === 'success';
            const detail = delivery.error || delivery.response_body || '-';
            const subjectID = this.formatDeliverySubject(delivery);
            return `
                <tr>
                    <td><span class="webhook-name">${this.escapeHtml(delivery.webhook_name || '-')}</span></td>
                    <td><span class="webhook-event">${this.escapeHtml(this.formatEventType(delivery.event_type))}</span></td>
                    <td><span class="webhook-delivery-status ${success ? 'is-success' : 'is-failed'}">${success ? '成功' : '失败'}</span></td>
                    <td>${delivery.http_status_code || '-'}</td>
                    <td>${this.escapeHtml(subjectID)}</td>
                    <td>${this.formatDuration(delivery.duration_ms)}</td>
                    <td>${delivery.attempt || '-'}</td>
                    <td><time class="webhook-time">${new Date(delivery.created_at).toLocaleString()}</time></td>
                    <td><span class="webhook-delivery-detail">${this.escapeHtml(this.truncateText(detail, 240))}</span></td>
                </tr>
            `;
        }).join('');
    }

    formatDeliverySubject(delivery) {
        if (delivery.sms_id) return `短信 #${delivery.sms_id}`;
        if (delivery.alert_id) return `告警 #${delivery.alert_id}`;
        return '-';
    }

    resetDeliveryPageAndList() {
        this.deliveryPage = 0;
        this.listDeliveries();
    }

    deliveryPrevPage() {
        if (this.deliveryPage > 0) {
            this.deliveryPage--;
            this.listDeliveries();
        }
    }

    deliveryNextPage() {
        const totalPages = Math.ceil(this.deliveryTotal / this.deliveryPageSize);
        if (this.deliveryPage < totalPages - 1) {
            this.deliveryPage++;
            this.listDeliveries();
        }
    }

    updateDeliveryPagination() {
        const totalPages = Math.max(1, Math.ceil(this.deliveryTotal / this.deliveryPageSize));
        const pageInfo = $('#webhookDeliveryPageInfo');
        const prevBtn = $('#webhookDeliveryPrevPageBtn');
        const nextBtn = $('#webhookDeliveryNextPageBtn');

        if (pageInfo) {
            pageInfo.textContent = `第 ${this.deliveryPage + 1} 页 / 共 ${totalPages} 页 (总计: ${this.deliveryTotal} 条)`;
        }
        if (prevBtn) {
            prevBtn.disabled = this.deliveryPage === 0;
        }
        if (nextBtn) {
            nextBtn.disabled = this.deliveryPage >= totalPages - 1;
        }
    }

    formatDuration(durationMs) {
        const value = Number(durationMs) || 0;
        if (value >= 1000) {
            return `${(value / 1000).toFixed(2)}s`;
        }
        return `${value}ms`;
    }

    truncateText(text, maxLength) {
        const value = String(text || '');
        if (value.length <= maxLength) return value;
        return value.slice(0, maxLength) + '...';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = String(text ?? '');
        return div.innerHTML;
    }
}
