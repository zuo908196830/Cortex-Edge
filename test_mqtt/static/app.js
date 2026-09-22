/**
 * MQTT Web Studio - Frontend Application Script
 */

document.addEventListener('DOMContentLoaded', () => {
  // --- DOM Element References ---
  const statusIndicator = document.getElementById('statusIndicator');
  const statusText = document.getElementById('statusText');
  const brokerInfoChip = document.getElementById('brokerInfoChip');
  const activeBrokerAddress = document.getElementById('activeBrokerAddress');

  // Stats Elements
  const statRxCount = document.getElementById('statRxCount');
  const statTxCount = document.getElementById('statTxCount');
  const statSubCount = document.getElementById('statSubCount');
  const logCountBadge = document.getElementById('logCountBadge');

  // Config Elements
  const protocolSelect = document.getElementById('protocolSelect');
  const brokerHost = document.getElementById('brokerHost');
  const brokerPort = document.getElementById('brokerPort');
  const clientIdInput = document.getElementById('clientId');
  const genClientIdBtn = document.getElementById('genClientIdBtn');
  const usernameInput = document.getElementById('username');
  const passwordInput = document.getElementById('password');
  const cleanSessionCheck = document.getElementById('cleanSession');
  const keepAliveInput = document.getElementById('keepAlive');
  const connectBtn = document.getElementById('connectBtn');
  const disconnectBtn = document.getElementById('disconnectBtn');
  const toggleConfigBtn = document.getElementById('toggleConfigBtn');
  const configCardBody = document.getElementById('configCardBody');

  // Subscription Elements
  const subForm = document.getElementById('subForm');
  const subTopicInput = document.getElementById('subTopic');
  const subQosSelect = document.getElementById('subQos');
  const subList = document.getElementById('subList');
  const clearSubsBtn = document.getElementById('clearSubsBtn');

  // Stream & Log Elements
  const feedBody = document.getElementById('feedBody');
  const emptyFeedTip = document.getElementById('emptyFeedTip');
  const messageStream = document.getElementById('messageStream');
  const searchInput = document.getElementById('searchInput');
  const directionFilter = document.getElementById('directionFilter');
  const autoScrollCheck = document.getElementById('autoScrollCheck');
  const clearLogsBtn = document.getElementById('clearLogsBtn');

  // Publisher Elements
  const pubTopicInput = document.getElementById('pubTopic');
  const pubQosSelect = document.getElementById('pubQos');
  const pubRetainCheck = document.getElementById('pubRetain');
  const templateSelect = document.getElementById('templateSelect');
  const pubPayloadTextarea = document.getElementById('pubPayload');
  const formatJsonBtn = document.getElementById('formatJsonBtn');
  const clearPayloadBtn = document.getElementById('clearPayloadBtn');
  const publishBtn = document.getElementById('publishBtn');
  const historyCountText = document.getElementById('historyCountText');

  // Toast Container
  const toastContainer = document.getElementById('toastContainer');

  // --- State Variables ---
  let ws = null;
  let isMqttConnected = false;
  let rxCount = 0;
  let txCount = 0;
  let logEntries = [];
  let subscribedTopics = new Map(); // topic -> { qos, color }
  let sentHistoryCount = 0;

  // Topic Color Palette Generator
  const tagColors = [
    '#00f2fe', '#4ade80', '#fbbf24', '#f472b6', 
    '#a855f7', '#38bdf8', '#ff7043', '#26a69a'
  ];
  let colorIndex = 0;

  function getNextColor() {
    const color = tagColors[colorIndex % tagColors.length];
    colorIndex++;
    return color;
  }

  // Generate random Client ID
  function generateClientId() {
    return 'mqtt_web_' + Math.random().toString(36).substring(2, 10);
  }

  clientIdInput.value = generateClientId();

  genClientIdBtn.addEventListener('click', () => {
    clientIdInput.value = generateClientId();
    showToast('已生成新 Client ID', 'info');
  });

  // --- Persistent Storage (Load Saved Config) ---
  function loadSavedConfig() {
    try {
      const savedConfig = localStorage.getItem('mqtt_studio_config');
      if (savedConfig) {
        const cfg = JSON.parse(savedConfig);
        if (cfg.protocol) protocolSelect.value = cfg.protocol;
        if (cfg.host) brokerHost.value = cfg.host;
        if (cfg.port) brokerPort.value = cfg.port;
        if (cfg.username) usernameInput.value = cfg.username;
      }
    } catch (e) {
      console.warn('Failed to load saved config:', e);
    }
  }

  function saveConfig() {
    const cfg = {
      protocol: protocolSelect.value,
      host: brokerHost.value,
      port: brokerPort.value,
      username: usernameInput.value
    };
    localStorage.setItem('mqtt_studio_config', JSON.stringify(cfg));
  }

  loadSavedConfig();

  // --- WebSocket Connection Handler (Bridge to Go Backend) ---
  function initWebSocketBridge() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws`;

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket Bridge connection established.');
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleBridgeMessage(msg);
      } catch (err) {
        console.error('Error parsing backend bridge message:', err);
      }
    };

    ws.onclose = () => {
      console.warn('WebSocket Bridge connection closed. Reconnecting in 3s...');
      setMqttState('disconnected', 'Bridge 断开');
      setTimeout(initWebSocketBridge, 3000);
    };

    ws.onerror = (err) => {
      console.error('WebSocket Bridge error:', err);
    };
  }

  initWebSocketBridge();

  function sendBridgeCmd(action, data = {}) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      showToast('后端 Bridge 服务未连接', 'error');
      return;
    }
    ws.send(JSON.stringify({ action, data }));
  }

  // Handle events received from Go Backend
  function handleBridgeMessage(msg) {
    const { event, data } = msg;

    switch (event) {
      case 'mqtt_connecting':
        setMqttState('connecting', '正在连接 Broker...');
        break;

      case 'mqtt_connected':
        isMqttConnected = true;
        setMqttState('connected', '已连接');
        activeBrokerAddress.textContent = `${data.server}`;
        brokerInfoChip.style.display = 'flex';
        connectBtn.style.display = 'none';
        disconnectBtn.style.display = 'inline-flex';
        showToast(`成功连接到 Broker (${data.server})`, 'success');
        saveConfig();
        break;

      case 'mqtt_disconnected':
        isMqttConnected = false;
        setMqttState('disconnected', '未连接');
        brokerInfoChip.style.display = 'none';
        connectBtn.style.display = 'inline-flex';
        disconnectBtn.style.display = 'none';
        showToast('已断开 MQTT Broker 连接', 'info');
        break;

      case 'mqtt_message':
        rxCount++;
        statRxCount.textContent = rxCount;
        addLogEntry({
          direction: 'rx',
          topic: data.topic,
          payload: data.payload,
          qos: data.qos,
          retain: data.retain,
          time: new Date().toLocaleTimeString()
        });
        break;

      case 'mqtt_published':
        txCount++;
        statTxCount.textContent = txCount;
        sentHistoryCount++;
        historyCountText.textContent = `发送历史: ${sentHistoryCount} 次`;
        addLogEntry({
          direction: 'tx',
          topic: data.topic,
          payload: data.payload,
          qos: data.qos,
          retain: data.retain,
          time: new Date().toLocaleTimeString()
        });
        showToast(`消息已成功发送至 ${data.topic}`, 'success');
        break;

      case 'mqtt_subscribed':
        addSubChip(data.topic, data.qos);
        showToast(`已成功订阅主题: ${data.topic}`, 'success');
        break;

      case 'mqtt_unsubscribed':
        removeSubChip(data.topic);
        showToast(`已取消订阅主题: ${data.topic}`, 'info');
        break;

      case 'mqtt_error':
        showToast(`MQTT 错误: ${data.message}`, 'error');
        if (!isMqttConnected) {
          setMqttState('disconnected', '连接失败');
        }
        break;

      default:
        console.log('Unhandled bridge event:', event, data);
    }
  }

  // Update Connection UI State
  function setMqttState(state, text) {
    statusIndicator.className = `status-badge ${state}`;
    statusText.textContent = text;
  }

  // --- Connect / Disconnect Handlers ---
  connectBtn.addEventListener('click', () => {
    const protocol = protocolSelect.value;
    const host = brokerHost.value.trim();
    const port = parseInt(brokerPort.value.trim(), 10);
    const clientId = clientIdInput.value.trim();
    const username = usernameInput.value.trim();
    const password = passwordInput.value.trim();
    const cleanSession = cleanSessionCheck.checked;
    const keepAlive = parseInt(keepAliveInput.value.trim(), 10) || 60;

    if (!host || !port) {
      showToast('请输入有效的 Host 地址和端口', 'error');
      return;
    }

    sendBridgeCmd('connect', {
      protocol,
      host,
      port,
      clientId,
      username,
      password,
      cleanSession,
      keepAlive
    });
  });

  disconnectBtn.addEventListener('click', () => {
    sendBridgeCmd('disconnect');
  });

  // Toggle Config Card Collapse
  toggleConfigBtn.addEventListener('click', () => {
    const isHidden = configCardBody.style.display === 'none';
    configCardBody.style.display = isHidden ? 'flex' : 'none';
    toggleConfigBtn.querySelector('i').className = isHidden ? 'fa-solid fa-chevron-up' : 'fa-solid fa-chevron-down';
  });

  // --- Subscription Management ---
  subForm.addEventListener('submit', (e) => {
    e.preventDefault();
    if (!isMqttConnected) {
      showToast('请先连接 MQTT Broker！', 'error');
      return;
    }

    const topic = subTopicInput.value.trim();
    const qos = parseInt(subQosSelect.value, 10);

    if (!topic) return;

    if (subscribedTopics.has(topic)) {
      showToast(`主题 ${topic} 已在订阅列表中`, 'info');
      return;
    }

    sendBridgeCmd('subscribe', { topic, qos });
  });

  function addSubChip(topic, qos) {
    const color = getNextColor();
    subscribedTopics.set(topic, { qos, color });
    statSubCount.textContent = subscribedTopics.size;

    renderSubList();
  }

  function removeSubChip(topic) {
    subscribedTopics.delete(topic);
    statSubCount.textContent = subscribedTopics.size;
    renderSubList();
  }

  function renderSubList() {
    subList.innerHTML = '';
    if (subscribedTopics.size === 0) {
      subList.innerHTML = '<div class="empty-sub-tip">暂无活动订阅，请输入主题并点击订阅</div>';
      clearSubsBtn.style.display = 'none';
      return;
    }

    clearSubsBtn.style.display = 'inline-block';

    subscribedTopics.forEach((info, topic) => {
      const item = document.createElement('div');
      item.className = 'sub-item';
      item.innerHTML = `
        <div class="sub-item-meta">
          <span class="sub-color-tag" style="background-color: ${info.color}"></span>
          <span class="sub-topic-name" title="${topic}">${topic}</span>
        </div>
        <div class="sub-item-actions">
          <span class="qos-tag">QoS ${info.qos}</span>
          <button class="btn-remove-sub" title="取消订阅" data-topic="${topic}">
            <i class="fa-solid fa-xmark"></i>
          </button>
        </div>
      `;

      item.querySelector('.btn-remove-sub').addEventListener('click', (e) => {
        const topicToUnsub = e.currentTarget.getAttribute('data-topic');
        sendBridgeCmd('unsubscribe', { topic: topicToUnsub });
      });

      // Quick click topic to fill publisher target topic
      item.querySelector('.sub-topic-name').addEventListener('click', () => {
        pubTopicInput.value = topic;
        showToast(`已填充目标 Topic: ${topic}`, 'info');
      });

      subList.appendChild(item);
    });
  }

  clearSubsBtn.addEventListener('click', () => {
    subscribedTopics.forEach((_, topic) => {
      sendBridgeCmd('unsubscribe', { topic });
    });
  });

  // --- Real-time Message Stream Logger ---
  function addLogEntry(entry) {
    logEntries.push(entry);
    logCountBadge.textContent = `${logEntries.length} 条`;

    if (emptyFeedTip.style.display !== 'none') {
      emptyFeedTip.style.display = 'none';
    }

    renderSingleLog(entry);
  }

  function renderSingleLog(entry) {
    // Apply Filters
    if (!matchesFilter(entry)) return;

    const msgCard = document.createElement('div');
    msgCard.className = 'msg-card';

    const isJson = isJsonString(entry.payload);
    const formattedContent = isJson 
      ? highlightJson(JSON.stringify(JSON.parse(entry.payload), null, 2))
      : escapeHtml(entry.payload);

    const topicColor = subscribedTopics.has(entry.topic)
      ? subscribedTopics.get(entry.topic).color
      : 'var(--accent-cyan)';

    msgCard.innerHTML = `
      <div class="msg-header">
        <div class="msg-meta-left">
          <span class="msg-dir-badge ${entry.direction}">${entry.direction.toUpperCase()}</span>
          <span class="msg-topic" style="border-left: 3px solid ${topicColor}">${escapeHtml(entry.topic)}</span>
          <span class="qos-tag">QoS ${entry.qos}</span>
          ${entry.retain ? '<span class="qos-tag" style="color:var(--accent-amber)">Retain</span>' : ''}
        </div>
        <div class="msg-meta-right">
          <span class="msg-time"><i class="fa-regular fa-clock"></i> ${entry.time}</span>
        </div>
      </div>
      <div class="msg-body">${formattedContent}</div>
      <div class="msg-actions">
        <button class="btn-xs-ghost btn-copy-payload" title="复制消息体">
          <i class="fa-regular fa-copy"></i> 复制 Payload
        </button>
        <button class="btn-xs-ghost btn-reuse-topic" title="填充到发送框">
          <i class="fa-solid fa-reply"></i> 重用 Topic
        </button>
      </div>
    `;

    // Action Handlers
    msgCard.querySelector('.btn-copy-payload').addEventListener('click', () => {
      navigator.clipboard.writeText(entry.payload);
      showToast('Payload 已复制到剪贴板', 'success');
    });

    msgCard.querySelector('.btn-reuse-topic').addEventListener('click', () => {
      pubTopicInput.value = entry.topic;
      pubPayloadTextarea.value = entry.payload;
      showToast('已复制 Topic 与 Payload 到发送区', 'info');
    });

    messageStream.appendChild(msgCard);

    if (autoScrollCheck.checked) {
      feedBody.scrollTop = feedBody.scrollHeight;
    }
  }

  function matchesFilter(entry) {
    const searchVal = searchInput.value.toLowerCase().trim();
    const dirVal = directionFilter.value;

    if (dirVal !== 'all' && entry.direction !== dirVal) {
      return false;
    }

    if (searchVal) {
      const matchTopic = entry.topic.toLowerCase().includes(searchVal);
      const matchPayload = entry.payload.toLowerCase().includes(searchVal);
      if (!matchTopic && !matchPayload) return false;
    }

    return true;
  }

  function reRenderAllLogs() {
    messageStream.innerHTML = '';
    const visibleEntries = logEntries.filter(matchesFilter);
    if (visibleEntries.length === 0 && logEntries.length > 0) {
      // Empty filter result
    } else if (logEntries.length === 0) {
      emptyFeedTip.style.display = 'flex';
      return;
    }

    emptyFeedTip.style.display = 'none';
    visibleEntries.forEach(renderSingleLog);
  }

  searchInput.addEventListener('input', reRenderAllLogs);
  directionFilter.addEventListener('change', reRenderAllLogs);

  clearLogsBtn.addEventListener('click', () => {
    logEntries = [];
    messageStream.innerHTML = '';
    emptyFeedTip.style.display = 'flex';
    logCountBadge.textContent = '0 条';
    showToast('已清空所有消息日志', 'info');
  });

  // --- Publisher Logic ---
  publishBtn.addEventListener('click', () => {
    if (!isMqttConnected) {
      showToast('请先连接 MQTT Broker！', 'error');
      return;
    }

    const topic = pubTopicInput.value.trim();
    const qos = parseInt(pubQosSelect.value, 10);
    const retain = pubRetainCheck.checked;
    const payload = pubPayloadTextarea.value;

    if (!topic) {
      showToast('请输入目标 Topic！', 'error');
      return;
    }

    sendBridgeCmd('publish', {
      topic,
      qos,
      retain,
      payload
    });
  });

  // Payload Templates
  templateSelect.addEventListener('change', (e) => {
    const val = e.target.value;
    switch (val) {
      case 'text':
        pubPayloadTextarea.value = 'Hello MQTT World from Edge Studio!';
        break;
      case 'sensor':
        pubPayloadTextarea.value = JSON.stringify({
          device_id: 'sensor_temp_01',
          temperature: (20 + Math.random() * 8).toFixed(2),
          humidity: (45 + Math.random() * 20).toFixed(2),
          timestamp: Date.now()
        }, null, 2);
        break;
      case 'status':
        pubPayloadTextarea.value = JSON.stringify({
          event: 'STATUS_UPDATE',
          status: 'ONLINE',
          battery: 98,
          uptime_seconds: 14200
        }, null, 2);
        break;
      case 'control':
        pubPayloadTextarea.value = JSON.stringify({
          command: 'SET_POWER',
          params: { power: 'ON', mode: 'AUTO', val: 100 }
        }, null, 2);
        break;
    }
    showToast('已加载快捷 Payload 模板', 'info');
  });

  formatJsonBtn.addEventListener('click', () => {
    const val = pubPayloadTextarea.value.trim();
    if (!val) return;
    try {
      const parsed = JSON.parse(val);
      pubPayloadTextarea.value = JSON.stringify(parsed, null, 2);
      showToast('JSON 格式化成功', 'success');
    } catch (e) {
      showToast('不是有效的 JSON 字符串', 'error');
    }
  });

  clearPayloadBtn.addEventListener('click', () => {
    pubPayloadTextarea.value = '';
  });

  // --- Utility Helper Functions ---
  function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    let iconClass = 'fa-circle-info';
    if (type === 'success') iconClass = 'fa-circle-check';
    if (type === 'error') iconClass = 'fa-circle-exclamation';

    toast.innerHTML = `
      <i class="fa-solid ${iconClass}"></i>
      <span>${escapeHtml(message)}</span>
    `;

    toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateX(20px)';
      toast.style.transition = 'all 0.25s ease';
      setTimeout(() => toast.remove(), 250);
    }, 3200);
  }

  function isJsonString(str) {
    if (typeof str !== 'string') return false;
    try {
      const result = JSON.parse(str);
      return typeof result === 'object' && result !== null;
    } catch (e) {
      return false;
    }
  }

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function highlightJson(jsonStr) {
    const escaped = escapeHtml(jsonStr);
    return escaped.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g, (match) => {
      let cls = 'json-number';
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = 'json-key';
        } else {
          cls = 'json-string';
        }
      } else if (/true|false/.test(match)) {
        cls = 'json-boolean';
      } else if (/null/.test(match)) {
        cls = 'json-null';
      }
      return `<span class="${cls}">${match}</span>`;
    });
  }
});
