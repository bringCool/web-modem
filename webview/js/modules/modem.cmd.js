/**
 * AT 命令快捷指令数据
 * 包含所有预设的AT命令选项，按功能分组
 */
export const AT_COMMANDS = [
    {
        label: "基本控制",
        options: [
            { value: "AT", text: "AT - 测试连接" },
            { value: "ATE0", text: "ATE0 - 关闭回显" },
            { value: "ATE1", text: "ATE1 - 开启回显" },
            { value: "ATZ", text: "ATZ - 重置 modem" },
            { value: "AT&F", text: "AT&F - 恢复出厂设置" },
            { value: "AT&W", text: "AT&W - 保存设置" }
        ]
    },
    {
        label: "设备身份信息",
        options: [
            { value: "AT+CGSN", text: "AT+CGSN - 查询 IMEI" },
            { value: "AT+CGMI", text: "AT+CGMI - 查询制造商" },
            { value: "AT+CGMM", text: "AT+CGMM - 查询型号" },
            { value: "AT+CGMR", text: "AT+CGMR - 查询版本" },
            { value: "AT+CIMI", text: "AT+CIMI - 查询 IMSI" },
            { value: "AT+CCID", text: "AT+CCID - 查询 ICCID" },
            { value: "AT+CNUM", text: "AT+CNUM - 查询本机号码" }
        ]
    },
    {
        label: "网络状态",
        options: [
            { value: "AT+COPS", text: "AT+COPS - 查询/设置运营商" },
            { value: "AT+CNMP", text: "AT+CNMP - 查询/设置网络模式" },
            { value: "AT+CREG", text: "AT+CREG - 查询网络注册状态" },
            { value: "AT+CGREG", text: "AT+CGREG - 查询 GPRS 注册状态" },
            { value: "AT+CEREG?", text: "AT+CEREG? - 查询 LTE/EPS 注册状态" },
            { value: "AT+CSQ", text: "AT+CSQ - 查询信号质量" },
            { value: "AT+CESQ", text: "AT+CESQ - 查询扩展信号质量" },
            { value: "AT+QCSQ", text: "AT+QCSQ - 查询 LTE 信号质量" },
            { value: "AT+QENG=\"servingcell\"", text: "AT+QENG - 查询服务小区" },
            { value: "AT+MUESTATS=\"radio\"", text: "AT+MUESTATS - 查询 ML307A 无线状态" }
        ]
    },
    {
        label: "SIM 卡管理",
        options: [
            { value: "AT+CPIN", text: "AT+CPIN - 查询 SIM 卡状态" },
            { value: "AT+CPWD", text: "AT+CPWD - 修改 PIN 码" },
            { value: "AT+CLCK", text: "AT+CLCK - 查询/设置 PIN 锁" }
        ]
    },
    {
        label: "设备状态",
        options: [
            { value: "AT+CBC", text: "AT+CBC - 查询电池电量" },
            { value: "AT+CPMUTEMP", text: "AT+CPMUTEMP - 查询设备温度" },
            { value: "AT+CCLK", text: "AT+CCLK - 查询/设置网络时间" }
        ]
    },
    {
        label: "网络配置",
        options: [
            { value: "AT+CGDCONT", text: "AT+CGDCONT - 查询/设置 APN" },
            { value: "AT+CGPADDR", text: "AT+CGPADDR - 查询 IP 地址" },
            { value: "AT+CGACT", text: "AT+CGACT - 查询/设置 PDP 上下文" }
        ]
    },
    {
        label: "短信相关",
        options: [
            { value: "AT+CMGF", text: "AT+CMGF - 查询/设置短信格式" },
            { value: "AT+CPMS", text: "AT+CPMS - 查询/设置短信存储位置" },
            { value: "AT+CSCA", text: "AT+CSCA - 查询/设置短信中心号码" },
            { value: "AT+CMGL", text: "AT+CMGL - 列出短信" },
            { value: "AT+CMGR", text: "AT+CMGR - 读取短信" },
            { value: "AT+CMGD", text: "AT+CMGD - 删除短信" },
            { value: "AT+CMGS", text: "AT+CMGS - 发送短信" }
        ]
    },
    {
        label: "语音通话",
        options: [
            { value: "ATD", text: "ATD - 拨号" },
            { value: "ATA", text: "ATA - 接听" },
            { value: "ATH", text: "ATH - 挂断" },
            { value: "AT+CLIP", text: "AT+CLIP - 查询/设置来电显示" },
            { value: "AT+CLCC", text: "AT+CLCC - 查询通话状态" },
            { value: "AT+CCWA", text: "AT+CCWA - 查询/设置呼叫等待" },
            { value: "AT+CCFC", text: "AT+CCFC - 查询/设置呼叫转移" }
        ]
    }
];
