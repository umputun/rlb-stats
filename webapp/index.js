import {getReadableDuration, loadData, createQueryParams, dateTimeOptions} from './data.js';
import {drawChart as drawAnyChart} from './anycharts.js';
import {drawChart as drawTauChart} from './taucharts.js';
import {drawChart as drawEChart} from './echarts.js';

// default charts are for last 24 hours
const defaultMinutes = 24 * 60;


const topFilesContainer = document.getElementById('top-files');
const topNodesContainer = document.getElementById('top-nodes');
const errorParagraph = document.getElementById("error");

const url = (new URL(location.href));
const chartLibrary = url.searchParams.get('chartLibrary');

function getDrawFunction() {
  const map = {
    any: drawAnyChart,
    tau: drawTauChart,
    e: drawEChart,
  };
  return map[chartLibrary] || drawTauChart;
}

const drawChart = getDrawFunction();

function getTopFilesTitle(date) {
  return `Top 10 files downloaded from ${date.toLocaleDateString(
    [],
    dateTimeOptions
  )}`
}

function getTopNodesTitle(date) {
  return `Top 10 nodes from ${date.toLocaleDateString(
    [],
    dateTimeOptions
  )}`
}

function setTitle(minutes) {
  document.title = `rlb-stats: data for the last ${getReadableDuration(
    minutes
  )}`;
}

async function drawPage(minutes, drawChart) {
  try {
    const queryParams = createQueryParams(minutes);
    setTitle(minutes);
    const data = await loadData(queryParams);
    drawChart({data: data.files, title: getTopFilesTitle(queryParams.from), container: topFilesContainer});
    drawChart({data: data.nodes, title: getTopNodesTitle(queryParams.from), container: topNodesContainer});
  } catch (e) {
    errorParagraph.innerHTML = e.message;
  }
}

const buttons = document.getElementById("period-buttons");
buttons.addEventListener(
  "click",
  event => {
    if (event.target.tagName === "BUTTON") {
      const minutes = event.target.dataset.minutes;
      drawPage(minutes, drawChart).catch((e) => {
        console.error(e);
      });
    }
  }
);

const chartsSelect = document.getElementById('chart-type');
chartsSelect.value = chartLibrary;

chartsSelect.addEventListener('change', (e) => {
  const value = e.target.value;
  const url = new URL(location.href);
  url.searchParams.set('chartLibrary', value);
  location.href = url.toString();
});


drawPage(defaultMinutes, drawChart).catch((e) => {
  console.error(e);
});
