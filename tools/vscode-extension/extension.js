const vscode = require('vscode');
const cp = require('child_process');

let statusBarItem;

/**
 * @param {vscode.ExtensionContext} context
 */
function activate(context) {
    // 1. Status Bar Indicator
    statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.command = 'agentmesh.diagnoseAgent';
    statusBarItem.text = '$(shield) AgentMesh: Active';
    statusBarItem.tooltip = 'AgentMesh Operational Intelligence & Confused Deputy Protection';
    statusBarItem.show();
    context.subscriptions.push(statusBarItem);

    // 2. Command: Inspect ADK Graph
    let inspectGraphCmd = vscode.commands.registerCommand('agentmesh.inspectGraph', async () => {
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders) {
            vscode.window.showErrorMessage('AgentMesh: Open an ADK project folder first.');
            return;
        }
        const rootPath = workspaceFolders[0].uri.fsPath;
        const cliPath = vscode.workspace.getConfiguration('agentmesh').get('cliPath') || 'agentmesh';

        vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: 'AgentMesh: Inspecting ADK Go Project...',
            cancellable: false
        }, () => {
            return new Promise((resolve) => {
                cp.exec(`${cliPath} adk graph inspect "${rootPath}" --json`, (err, stdout, stderr) => {
                    if (err) {
                        vscode.window.showErrorMessage(`AgentMesh inspection failed: ${stderr || err.message}`);
                        resolve();
                        return;
                    }
                    try {
                        const graphData = JSON.parse(stdout);
                        showGraphWebview(context, graphData);
                    } catch (parseErr) {
                        vscode.window.showInformationMessage(stdout);
                    }
                    resolve();
                });
            });
        });
    });

    // 3. Command: Simulate Policy & Check Confused Deputy
    let simulatePolicyCmd = vscode.commands.registerCommand('agentmesh.simulatePolicy', async () => {
        const cliPath = vscode.workspace.getConfiguration('agentmesh').get('cliPath') || 'agentmesh';
        cp.exec(`${cliPath} policy simulate`, (err, stdout) => {
            if (err) {
                vscode.window.showErrorMessage(`AgentMesh policy simulation error: ${err.message}`);
                return;
            }
            vscode.window.showInformationMessage(`AgentMesh Policy Verdict:\n${stdout}`);
        });
    });

    // 4. Command: Run Operational Diagnostics
    let diagnoseCmd = vscode.commands.registerCommand('agentmesh.diagnoseAgent', async () => {
        const cliPath = vscode.workspace.getConfiguration('agentmesh').get('cliPath') || 'agentmesh';
        cp.exec(`${cliPath} doctor`, (err, stdout) => {
            const channel = vscode.window.createOutputChannel('AgentMesh Diagnostics');
            channel.clear();
            channel.appendLine(stdout);
            channel.show();
        });
    });

    // 5. Command: Run Adversarial Red-Team Evaluator
    let redTeamCmd = vscode.commands.registerCommand('agentmesh.evalRedTeam', async () => {
        const cliPath = vscode.workspace.getConfiguration('agentmesh').get('cliPath') || 'agentmesh';
        vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: 'AgentMesh: Executing LLM Red-Team Probes...',
            cancellable: false
        }, () => {
            return new Promise((resolve) => {
                cp.exec(`${cliPath} eval redteam --json`, (err, stdout) => {
                    const channel = vscode.window.createOutputChannel('AgentMesh Red-Team Report');
                    channel.clear();
                    channel.appendLine(stdout);
                    channel.show();
                    resolve();
                });
            });
        });
    });

    context.subscriptions.push(inspectGraphCmd, simulatePolicyCmd, diagnoseCmd, redTeamCmd);
}

function showGraphWebview(context, graphData) {
    const panel = vscode.window.createWebviewPanel(
        'agentmeshGraph',
        `AgentGraph: ${graphData.graphId || 'ADK Graph'}`,
        vscode.ViewColumn.Beside,
        { enableScripts: true }
    );

    const nodesHtml = (graphData.nodes || []).map(n => `
        <li style="margin: 6px 0; padding: 6px; background: #252526; border-radius: 4px;">
            <strong>${n.name || n.id}</strong> 
            <span style="color: #4ec9b0;">[${n.type}]</span>
            <div style="color: #9cdcfe; font-size: 11px;">Target: ${n.target || 'self'}</div>
        </li>
    `).join('');

    panel.webview.html = `<!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>AgentMesh Graph</title>
        <style>
            body { font-family: sans-serif; padding: 16px; color: #ccc; background: #1e1e1e; }
            h2 { color: #569cd6; }
            ul { list-style: none; padding: 0; }
        </style>
    </head>
    <body>
        <h2>AgentMesh ADK Topology Inspector</h2>
        <p>Canonical Graph ID: <code>${graphData.graphId || 'adk-project'}</code></p>
        <p>Entrypoint: <code>${graphData.entrypoint || 'default'}</code></p>
        <h3>Discovered Nodes (${(graphData.nodes || []).length}):</h3>
        <ul>${nodesHtml}</ul>
    </body>
    </html>`;
}

function deactivate() {
    if (statusBarItem) {
        statusBarItem.dispose();
    }
}

module.exports = {
    activate,
    deactivate
};
