async function api(path, options = {}) {
  const res = await fetch(`/api${path}`, options);
  if (!res.ok) {
    let msg = "请求失败";
    try {
      const j = await res.json();
      msg = j.error || msg;
    } catch (_) {}
    throw new Error(msg);
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return res.json();
  return res.blob();
}

function selectedGroupId() {
  return Number(document.getElementById("groupSelect").value || 0);
}

async function init() {
  await loadGroups();
  await loadEmployees();
}

async function loadGroups() {
  const groups = await api("/groups");
  const sel = document.getElementById("groupSelect");
  sel.innerHTML = groups.map(g => `<option value="${g.id}">${g.department}-${g.name}</option>`).join("");
}

async function createGroup() {
  try {
    await api("/groups", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        name: document.getElementById("groupName").value,
        department: document.getElementById("deptName").value
      })
    });
    await loadGroups();
    alert("新增小组成功");
  } catch (e) { alert(e.message); }
}

async function createEmployee() {
  try {
    await api("/employees", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        name: document.getElementById("employeeName").value,
        role: document.getElementById("employeeRole").value,
        category: document.getElementById("employeeCategory").value,
        canNight: document.getElementById("canNight").checked,
        active: true
      })
    });
    await loadEmployees();
    alert("新增员工成功");
  } catch (e) { alert(e.message); }
}

async function deleteEmployee(id) {
  if (!confirm("确定删除该员工吗？")) return;
  await api(`/employees/${id}`, {method:"DELETE"});
  await loadEmployees();
}

async function loadEmployees() {
  try {
    const gid = selectedGroupId();
    if (!gid) return;
    const items = await api(`/employees?groupId=${gid}`);
    const tbody = document.querySelector("#employeeTable tbody");
    tbody.innerHTML = items.map(e => `
      <tr>
        <td>${e.id}</td>
        <td>${e.name}</td>
        <td>${e.role}</td>
        <td>${e.category || ""}</td>
        <td>${e.canNight ? "是":"否"}</td>
        <td><button type="button" class="btn btn--danger" onclick="deleteEmployee(${e.id})">删除</button></td>
      </tr>`).join("");
  } catch (e) { alert(e.message); }
}

document.getElementById("groupSelect").addEventListener("change", loadEmployees);

async function createPost() {
  try {
    await api("/posts", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        name: document.getElementById("postName").value,
        required: Number(document.getElementById("postRequired").value),
        priority: Number(document.getElementById("postPriority").value),
        enabled: true
      })
    });
    alert("新增岗位成功");
  } catch (e) { alert(e.message); }
}

async function importNight() {
  try {
    const fileInput = document.getElementById("nightFile");
    if (!fileInput.files.length) throw new Error("请先选择文件");
    const fd = new FormData();
    fd.append("month", document.getElementById("monthNight").value);
    fd.append("file", fileInput.files[0]);
    await api("/night-shifts/import", { method: "POST", body: fd });
    alert("夜班导入成功");
  } catch (e) { alert(e.message); }
}

async function saveConstraint() {
  try {
    await api("/constraints", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        month: document.getElementById("monthConstraint").value,
        role: document.getElementById("constraintRole").value,
        restDaysGoal: Number(document.getElementById("constraintRestDays").value)
      })
    });
    alert("休息目标已保存");
  } catch (e) { alert(e.message); }
}

async function createRule() {
  try {
    await api("/rules", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        name: "special-rule",
        ruleType: document.getElementById("ruleType").value,
        dayOfMonth: Number(document.getElementById("ruleDayOfMonth").value || 0),
        weekday: Number(document.getElementById("ruleWeekday").value || 0),
        postName: document.getElementById("rulePostName").value,
        required: Number(document.getElementById("ruleRequired").value),
        enabled: true
      })
    });
    alert("特殊规则已保存");
  } catch (e) { alert(e.message); }
}

async function generateSchedule() {
  try {
    await api("/schedule/generate", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        month: document.getElementById("monthSchedule").value
      })
    });
    await loadSchedule();
    alert("排班生成成功");
  } catch (e) { alert(e.message); }
}

async function loadSchedule() {
  try {
    const gid = selectedGroupId();
    const month = document.getElementById("monthSchedule").value;
    const items = await api(`/schedule?groupId=${gid}&month=${month}`);
    const tbody = document.querySelector("#scheduleTable tbody");
    tbody.innerHTML = items.map(i => `<tr><td>${i.day}</td><td>${i.employee}</td><td>${i.shiftName}</td></tr>`).join("");
  } catch (e) { alert(e.message); }
}

async function exportSchedule() {
  try {
    const gid = selectedGroupId();
    const month = document.getElementById("monthSchedule").value;
    window.open(`/api/schedule/export?groupId=${gid}&month=${month}`, "_blank");
  } catch (e) { alert(e.message); }
}

window.createGroup = createGroup;
window.createEmployee = createEmployee;
window.deleteEmployee = deleteEmployee;
window.createPost = createPost;
window.importNight = importNight;
window.saveConstraint = saveConstraint;
window.createRule = createRule;
window.generateSchedule = generateSchedule;
window.loadSchedule = loadSchedule;
window.exportSchedule = exportSchedule;

init();
