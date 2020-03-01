loadAndDrawTau();

async function loadDataTau(args) {
  const url = `/api/candle?from=${args.from.toISOString()}&to=${args.to.toISOString()}`;
  const errorParagraph = document.getElementById("error");
  errorParagraph.innerHTML = "";
  try {
    const response = await fetch(url);
    if (!response.ok) {
      errorParagraph.innerHTML = `Error retrieving data for graphs: ${response.status} ${response.statusText}`;
      return [];
    }
    return await response.json();
  } catch (error) {
    errorParagraph.innerHTML = `Error retrieving data for graphs: ${error.message}`;
    return [];
  }
}

function transformToFilesTau(jsonData) {
  if (jsonData.length === 0) {
    return { files: [], nodes: [] };
  }
  const filesData = jsonData.reduce(
    (acc, element) => {
      const entryDate = new Date(element.StartMinute).getTime();

      // all files volume to given minute
      acc.files.push({
        file: "all",
        date: entryDate,
        volume: element.Nodes.all.Volume
      });

      // per-file stats for given minute
      Object.keys(element.Nodes.all.Files).forEach(file => {
        acc.files.push({
          file: file,
          date: entryDate,
          volume: element.Nodes.all.Files[file]
        });
      });

      // per-node stats for given minute
      Object.keys(element.Nodes).forEach(node => {
        acc.nodes.push({
          node: node,
          date: entryDate,
          volume: element.Nodes[node].Volume
        });
      });

      return acc;
    },
    { files: [], nodes: [] }
  );
  return {
    files: filesData.files,
    nodes: filesData.nodes
  };
}

// date and time format for title and tooltips
const dateTimeOptionsTau = {
  year: "numeric",
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit"
};

// default charts are for last 24 hours
async function loadAndDrawTau(minutes = 24 * 60) {
  document.title = `rlb-stats: data for the last ${getReadableDurationTau(
    minutes
  )}`;
  const args = {
    from: new Date(new Date().setMinutes(new Date().getMinutes() - minutes)),
    to: new Date()
  };
  const jsonData = await loadDataTau(args);
  const data = transformToFilesTau(jsonData);
  drawChartTau(
    data.files,
    "tau-top-files",
    `Top 10 files downloaded from ${args.from.toLocaleDateString(
      [],
      dateTimeOptionsTau
    )}`
  );
  drawChartTau(
    data.nodes,
    "tau-top-nodes",
    `Top 10 nodes from ${args.from.toLocaleDateString([], dateTimeOptions)}`
  );
}

const buttonsTau = document.getElementById("period-buttons");
buttonsTau.addEventListener("click", event =>
  loadAndDrawTau(parseInt(event.target.dataset.minutes))
);

function getReadableDurationTau(minutes) {
  let tempMinutes = minutes;
  let readableDate = "";
  if (tempMinutes >= 60 * 24) {
    readableDate += String(Math.floor(tempMinutes / (60 * 24))) + "d";
    tempMinutes = tempMinutes % (60 * 24);
  }
  if (tempMinutes >= 60) {
    readableDate += String(Math.floor(tempMinutes / 60)) + "h";
    tempMinutes = tempMinutes % 60;
  }
  if (tempMinutes > 0) {
    readableDate += `${tempMinutes}m`;
  }
  return readableDate;
}

async function drawChartTau(data, container, title) {
  const lineName = data[0].file ? "file" : "node";
  //clear the container
  document.getElementById(container).innerHTML = "";
  new Taucharts.Chart({
    data: data,
    type: "line",
    x: "date",
    y: "volume",
    color: lineName,
    guide: {
      x: { nice: false }
    },
    plugins: [
      Taucharts.api.plugins.get("legend")({
        position: "bottom"
      }),
      Taucharts.api.plugins.get("tooltip")()
    ]
  }).renderTo(`#${container}`);
}
