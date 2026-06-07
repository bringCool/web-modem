/**
 * Modem 调测系统主入口
 * 负责初始化所有模块并管理应用生命周期
 */

import { Logger } from './modules/logger.js';
import { UIrender } from './utils/dom.js';

import { ModemManager } from './modules/modem.js';
import { SmsdbManager } from './modules/smsdb.js';
import { WebhookManager } from './modules/webhook.js';
import { AlertManager } from './modules/alert.js';
import { SettingManager } from './modules/setting.js';
import { UpdateManager } from './modules/update.js';

import { TabManager } from './modules/tabs.js';

import { SseService } from './modules/sse.js';

// 全局应用对象
window.app = {};

/**
 * 应用初始化
 * 初始化所有管理器模块并设置全局应用对象
 */
document.addEventListener('DOMContentLoaded', async () => {
    try {
        // 初始化全局日志
        app.logger = new Logger();

        // 初始化全局渲染器
        app.render = new UIrender();

        // 初始化功能管理器
        app.modemManager = new ModemManager();
        app.smsdbManager = new SmsdbManager();
        app.webhookManager = new WebhookManager();
        app.alertManager = new AlertManager();
        app.settingManager = new SettingManager();
        app.updateManager = new UpdateManager();

        // 初始化标签管理器
        app.tabManager = new TabManager();
        await app.modemManager.ready;
        app.tabManager.loadTabData();

        // 初始化 SSE 服务
        app.sse = new SseService();

        // 记录应用启动日志
        app.logger.success('Modem 调测系统已启动');
    } catch (error) {
        console.error('应用初始化失败:', error);
    }
});
