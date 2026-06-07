import { $, $$ } from '../utils/dom.js';
import { apiRequest, buildQueryString } from '../utils/api.js';

/**
 * 短信存储管理器类
 * 负责管理数据库中的短信数据，包括增删改查、分页、筛选等功能
 */
export class SmsdbManager {
    /**
     * 构造函数
     * 初始化短信存储管理器的基本状态和属性
     */
    constructor() {
        this.page = 0;                    // 当前页码
        this.pageSize = 50;               // 每页显示数量
        this.total = 0;                   // 总记录数
        this.selectedSmsdb = new Set();   // 选中的短信ID集合
    }

    /* =========================================
       短信存储管理 (Database SMS Management)
       ========================================= */

    /**
     * 列出短信存储
     * 根据分页和筛选条件获取短信列表
     */
    async listSmsdb() {
        try {
            const filter = {
                limit: this.pageSize,
                offset: this.page * this.pageSize
            };

            const deviceKey = $('#smsdbFilterDeviceKey')?.value.trim();
            if (deviceKey) {
                filter.device_key = deviceKey;
            }

            // 添加过滤条件
            const sendNumber = $('#smsdbFilterSendNumber')?.value.trim();
            if (sendNumber) {
                filter.send_number = sendNumber;
            }

            const direction = $('#smsdbFilterDirection')?.value;
            if (direction) {
                filter.direction = direction;
            }

            const startDate = $('#smsdbFilterStartDate')?.value;
            if (startDate) {
                filter.start_time = new Date(startDate).toISOString();
            }

            const endDate = $('#smsdbFilterEndDate')?.value;
            if (endDate) {
                const end = new Date(endDate);
                end.setHours(23, 59, 59, 999);
                filter.end_time = end.toISOString();
            }

            const queryString = buildQueryString(filter);
            const result = await apiRequest(`/smsdb/list?${queryString}`);

            this.total = Number(result?.total) || 0;
            this.displaySmsList(Array.isArray(result?.data) ? result.data : []);
            this.updateSmsdbPagination();
        } catch (error) {
            app.logger.error('加载短信存储失败: ' + error);
        }
    }

    /**
     * 显示短信存储列表
     * 将短信数据渲染到表格中
     * @param {Array} smsList - 短信列表数据
     */
    displaySmsList(smsList) {
        const tbody = $('#smsdbList');
        if (!tbody) return;

        if (!smsList || smsList.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="empty-table-cell">暂无短信</td></tr>';
            return;
        }

        tbody.innerHTML = smsList.map(sms => app.render.render('smsdbItem', {
            id: sms.id,
            direction: sms.direction === 'in' ? '接收' : '发送',
            send_number: sms.send_number || '-',
            receive_number: sms.receive_number || '-',
            device: this.getSmsDeviceLabel(sms),
            content: sms.content,
            receive_time: new Date(sms.receive_time).toLocaleString(),
            sms_ids: sms.sms_ids
        })).join('');
    }

    getSmsDeviceLabel(sms) {
        const number = sms.direction === 'out' ? sms.send_number : sms.receive_number;
        const value = [sms.device_imei, number, sms.modem_name].find(item => {
            const text = String(item || '').trim();
            return text && text !== '-' && text.toLowerCase() !== 'unknown';
        });
        if (value) return value;
        return '-';
    }

    async listRecentSmsdb(limit = 6) {
        const container = $('#recentSmsList');
        if (!container) return;

        const deviceKeys = app.modemManager?.getSmsIdentityKeys?.() || [];
        if (deviceKeys.length === 0) {
            container.innerHTML = '<div class="empty-state">请选择串口设备</div>';
            return;
        }

        container.innerHTML = this.recentSmsSkeleton(limit);
        try {
            const queryString = buildQueryString({ limit, offset: 0, device_key: deviceKeys });
            const result = await apiRequest(`/smsdb/list?${queryString}`);
            const smsList = Array.isArray(result?.data) ? result.data : [];

            if (smsList.length === 0) {
                container.innerHTML = '<div class="empty-state">暂无短信记录</div>';
                return;
            }

            container.innerHTML = smsList.map(sms => `
                <div class="recent-sms-item">
                    <div class="recent-sms-meta">
                        <span>${sms.direction === 'in' ? '接收' : '发送'}</span>
                        <span>${this.escapeHtml(sms.send_number || '-')}</span>
                        <span>${this.escapeHtml(sms.receive_number || sms.device_imei || sms.modem_name || '-')}</span>
                        <time>${new Date(sms.receive_time).toLocaleString()}</time>
                    </div>
                    <div class="recent-sms-content">${this.escapeHtml(sms.content || '')}</div>
                </div>
            `).join('');
        } catch (error) {
            container.innerHTML = '<div class="empty-state">短信记录加载失败</div>';
            app.logger.error('加载最近短信失败: ' + error);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    recentSmsSkeleton(count) {
        return Array.from({ length: Math.min(count, 4) }, () => `
            <div class="recent-sms-item recent-sms-skeleton">
                <div class="recent-sms-meta">
                    <span class="skeleton-line skeleton-chip"></span>
                    <span class="skeleton-line skeleton-meta"></span>
                    <span class="skeleton-line skeleton-meta"></span>
                </div>
                <div class="skeleton-line skeleton-message"></div>
            </div>
        `).join('');
    }

    toggleSmsdbSelection(id) {
        if (this.selectedSmsdb.has(id)) {
            this.selectedSmsdb.delete(id);
        } else {
            this.selectedSmsdb.add(id);
        }
    }

    toggleCheckAll() {
        const checkAll = $('#smsdbCheckAll');
        if (!checkAll) return;

        const checkboxes = $$('#smsdbList input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = checkAll.checked;
            const id = parseInt(checkbox.value);
            if (checkAll.checked) {
                this.selectedSmsdb.add(id);
            } else {
                this.selectedSmsdb.delete(id);
            }
        });
    }

    async deleteSmsdb(id) {
        if (!confirm('确定要删除这条短信吗？')) {
            return;
        }

        try {
            await apiRequest('/smsdb/delete', 'POST', { ids: [id] });
            app.logger.success('短信删除成功');
            this.listSmsdb();
        } catch (error) {
            app.logger.error('删除短信失败: ' + error);
        }
    }

    async copySmsdb(content) {
        try {
            // 方法1: 使用 Clipboard API (现代浏览器)
            if (navigator.clipboard && navigator.clipboard.writeText) {
                await navigator.clipboard.writeText(content);
                app.logger.success('短信内容已复制到剪贴板');
                return;
            }
            // 方法2: 使用传统的 document.execCommand (兼容性更好)
            const textArea = document.createElement('textarea');
            textArea.value = content;
            textArea.style.position = 'fixed';
            textArea.style.left = '-9999px';
            textArea.style.top = '0';
            document.body.appendChild(textArea);
            textArea.focus();
            textArea.select();
            if (document.execCommand('copy')) {
                app.logger.success('短信内容已复制到剪贴板');
            } else {
                app.logger.error('复制失败，请手动复制');
            }
            document.body.removeChild(textArea);
        } catch (error) {
            app.logger.error('复制失败: ' + error.message);
        }
    }

    async deleteSelectedSmsdb() {
        if (this.selectedSmsdb.size === 0) {
            app.logger.error('请先选择要删除的短信');
            return;
        }

        if (!confirm(`确定要删除选中的 ${this.selectedSmsdb.size} 条短信吗？`)) {
            return;
        }

        try {
            const ids = Array.from(this.selectedSmsdb);
            await apiRequest('/smsdb/delete', 'POST', { ids });
            app.logger.success(`成功删除 ${ids.length} 条短信`);
            this.selectedSmsdb.clear();
            this.listSmsdb();
        } catch (error) {
            app.logger.error('批量删除短信失败: ' + error);
        }
    }

    smsdbPrevPage() {
        if (this.page > 0) {
            this.page--;
            this.listSmsdb();
        }
    }

    smsdbNextPage() {
        const totalPages = Math.ceil(this.total / this.pageSize);
        if (this.page < totalPages - 1) {
            this.page++;
            this.listSmsdb();
        }
    }

    updateSmsdbPagination() {
        const totalPages = Math.ceil(this.total / this.pageSize);
        const pageInfo = $('#smsdbPageInfo');
        const prevBtn = $('#smsdbPrevPageBtn');
        const nextBtn = $('#smsdbNextPageBtn');

        if (pageInfo) {
            pageInfo.textContent = `第 ${this.page + 1} 页 / 共 ${totalPages} 页 (总计: ${this.total} 条)`;
        }

        if (prevBtn) {
            prevBtn.disabled = this.page === 0;
        }

        if (nextBtn) {
            nextBtn.disabled = this.page >= totalPages - 1;
        }
    }

    /* =========================================
       短信同步 (SMS Synchronization)
       ========================================= */

    /**
     * 同步所有已连接Modem短信到数据库
     */
    async syncAllModemSms() {
        try {
            app.logger.info('正在同步所有已连接卡的短信...');
            const result = await apiRequest('/smsdb/sync', 'POST', {});
            const results = Array.isArray(result?.results) ? result.results : [];

            if (results.length === 0) {
                app.logger.info('没有可同步的已连接卡');
                return;
            }

            results.forEach(item => this.logSyncResult(item));

            const newCount = Number(result?.newCount) || 0;
            const totalCount = Number(result?.totalCount) || 0;
            const failedCount = Number(result?.failedCount) || 0;

            if (failedCount > 0) {
                app.logger.error(`短信同步完成，${failedCount} 个卡失败`);
            } else if (newCount > 0) {
                app.logger.success(`全部卡同步完成，新增 ${newCount} 条短信 (共 ${totalCount} 条)`);
            } else {
                app.logger.info(`全部卡无新短信 (共 ${totalCount} 条)`);
            }

            if (newCount > 0) {
                await this.listSmsdb();
                await this.listRecentSmsdb();
            }
        } catch (error) {
            app.logger.error('同步全部短信失败: ' + error);
        }
    }

    /**
     * 兼容旧入口：短信记录是全局模块，默认同步所有卡。
     */
    async syncCurrentModemSms() {
        await this.syncAllModemSms();
    }

    /**
     * 同步指定Modem的短信到数据库
     * @param {string} modemName - Modem名称
     */
    async syncModemSms(modemName) {
        try {
            app.logger.info(`正在同步 ${modemName} 的短信...`);
            const result = await apiRequest('/smsdb/sync', 'POST', { name: modemName });

            this.logSyncResult(result);
            if ((Number(result?.newCount) || 0) > 0) {
                await this.listSmsdb();
                await this.listRecentSmsdb();
            }
        } catch (error) {
            app.logger.error(`同步 ${modemName} 短信失败: ` + error);
        }
    }

    logSyncResult(result) {
        const modemName = result?.modemName || '未知卡';
        const newCount = Number(result?.newCount) || 0;
        const totalCount = Number(result?.totalCount) || 0;

        if (result?.error) {
            app.logger.error(`[${modemName}] ${result.error}`);
        } else if (newCount > 0) {
            app.logger.success(`[${modemName}] 同步 ${newCount} 条新短信 (共 ${totalCount} 条)`);
        } else {
            app.logger.info(`[${modemName}] 无新短信 (共 ${totalCount} 条)`);
        }
    }
}
