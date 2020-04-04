// date and time format for title and tooltips
export const dateTimeOptions = {
  year: "numeric",
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit"
};

export const createQueryParams = (minutes) => {
  return {
    from: new Date(new Date().setMinutes(new Date().getMinutes() - minutes)),
    to: new Date()
  };
};

function makeError(message) {
  return new Error(`Error retrieving data for graphs: ${message}`)
}

function sortFilesOrNodes(arr) {
  return arr.sort((a, b) => {
    return (
      b.data.reduce((acc, element) => {
        return acc + element[1];
      }, 0) -
      a.data.reduce((acc, element) => {
        return acc + element[1];
      }, 0)
    );
  });
}

async function fetchData(queryParams) {
  const url = `api/candle?from=${queryParams.from.toISOString()}&to=${queryParams.to.toISOString()}`;
  try {
    const response = await fetch(url);
    if (!response.ok) {
      const error = makeError(`${response.status} ${response.statusText}`);
      error.httpError = true;
      throw error;
    }
    return await response.json();
  } catch (error) {
    if (error.httpError) {
      throw error;
    }
    throw makeError(error.message);
  }
}

function transformToFiles(jsonData) {
  if (jsonData.length === 0) {
    return {files: [], nodes: []};
  }
  const filesData = jsonData.reduce(
    (acc, element) => {
      const entryDate = new Date(element.StartMinute).getTime();

      // all files volume to given minute
      acc.files[0].data.push([entryDate, element.Nodes.all.Volume]);

      // per-file stats for given minute
      Object.keys(element.Nodes.all.Files).forEach(file => {
        if (!acc.files.find(el => el.file === file)) {
          acc.files.push({
            file: file,
            data: [[entryDate, element.Nodes.all.Files[file]]]
          });
        } else {
          acc.files.forEach(el => {
            if (el.file === file) {
              el.data.push([entryDate, element.Nodes.all.Files[file]]);
            }
          });
        }
      });

      // per-node stats for given minute
      Object.keys(element.Nodes).forEach(node => {
        if (!acc.nodes.find(el => el.node === node)) {
          acc.nodes.push({
            node: node,
            data: [[entryDate, element.Nodes[node].Volume]]
          });
        } else {
          acc.nodes.forEach(el => {
            if (el.node === node) {
              el.data.push([entryDate, element.Nodes[node].Volume]);
            }
          });
        }
      });

      return acc;
    },
    {files: [{file: "all", data: []}], nodes: []}
  );

  const sortedFilesData = sortFilesOrNodes(filesData.files);
  const sortedNodesData = sortFilesOrNodes(filesData.nodes);
  // choose only top 10 files or nodes and total qty
  return {
    files: sortedFilesData.slice(0, 11),
    nodes: sortedNodesData.slice(0, 11)
  };
}


export function getReadableDuration(minutes) {
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

export async function loadData(args) {
  const data = await fetchData(args);
  return transformToFiles(data);
}
