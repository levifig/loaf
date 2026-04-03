const command = `gh pr create --title "feat: add 'special' feature"`;
const m = command.match(/--title(?:\s+|=)(?:["']([^"']+)["']|([^\s"']+))/);
console.log(m ? (m[1] || m[2]) : "no match");
