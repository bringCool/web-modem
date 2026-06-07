/**
 * SSE服务类
 * 负责管理服务端事件流连接和事件分发
 */
export class SseService {
    /**
     * 构造函数
     */
    constructor() {
        this.source = null;
        this.eventListeners = new Map();
        this.url = '/events/modem';
        this.connect();
    }

    /**
     * 连接SSE事件流
     */
    connect() {
        if (this.source?.readyState === EventSource.OPEN || this.source?.readyState === EventSource.CONNECTING) {
            return;
        }

        try {
            this.source = new EventSource(this.url);
            this.setupEventListeners();
        } catch (error) {
            app.logger.error('SSE 连接失败:', error);
            this.emit('error', error);
        }
    }

    /**
     * 断开连接
     */
    disconnect() {
        if (this.source) {
            this.source.close();
            this.source = null;
        }
    }

    /**
     * 设置事件监听器
     */
    setupEventListeners() {
        this.source.onopen = async () => {
            app.logger.success('SSE 已连接');
            this.emit('connected');
        };

        this.source.onmessage = async (event) => {
            app.logger.info('SSE 消息:', event.data);
            this.emit('message', event);
        };

        this.source.onerror = async (error) => {
            app.logger.error('SSE 错误，正在自动重连');
            this.emit('error', error);
        };
    }

    /**
     * 添加事件监听器
     * @param {string} event - 事件名称
     * @param {Function} callback - 回调函数
     */
    on(event, callback) {
        if (!this.eventListeners.has(event)) {
            this.eventListeners.set(event, []);
        }
        this.eventListeners.get(event).push(callback);
    }

    /**
     * 移除事件监听器
     * @param {string} event - 事件名称
     * @param {Function} callback - 回调函数
     */
    off(event, callback) {
        if (this.eventListeners.has(event)) {
            const listeners = this.eventListeners.get(event);
            const index = listeners.indexOf(callback);
            if (index > -1) {
                listeners.splice(index, 1);
            }
        }
    }

    /**
     * 触发事件
     * @param {string} event - 事件名称
     * @param {any} data - 事件数据
     */
    emit(event, data) {
        if (this.eventListeners.has(event)) {
            const listeners = this.eventListeners.get(event);
            for (const callback of listeners) {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`SSE事件处理错误 (${event}):`, error);
                }
            }
        }
    }
}
